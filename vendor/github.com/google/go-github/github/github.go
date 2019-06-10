// Copyright 2013 The go-github AUTHORS. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:generate go run gen-accessors.go

package github

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/go-querystring/query"
)

const (
	defaultBaseURL = "https://api.github.com/"
	uploadBaseURL  = "https://uploads.github.com/"
	userAgent      = "go-github"

	headerRateLimit     = "X-RateLimit-Limit"
	headerRateRemaining = "X-RateLimit-Remaining"
	headerRateReset     = "X-RateLimit-Reset"
	headerOTP           = "X-GitHub-OTP"

	mediaTypeV3                = "application/vnd.github.v3+json"
	defaultMediaType           = "application/octet-stream"
	mediaTypeV3SHA             = "application/vnd.github.v3.sha"
	mediaTypeV3Diff            = "application/vnd.github.v3.diff"
	mediaTypeV3Patch           = "application/vnd.github.v3.patch"
	mediaTypeOrgPermissionRepo = "application/vnd.github.v3.repository+json"

	// Media Type values to access preview APIs

	// https://developer.github.com/changes/2014-12-09-new-attributes-for-stars-api/
	mediaTypeStarringPreview = "application/vnd.github.v3.star+json"

	// https://help.github.com/enterprise/2.4/admin/guides/migrations/exporting-the-github-com-organization-s-repositories/
	mediaTypeMigrationsPreview = "application/vnd.github.wyandotte-preview+json"

	// https://developer.github.com/changes/2016-04-06-deployment-and-deployment-status-enhancements/
	mediaTypeDeploymentStatusPreview = "application/vnd.github.ant-man-preview+json"

	// https://developer.github.com/changes/2016-02-19-source-import-preview-api/
	mediaTypeImportPreview = "application/vnd.github.barred-rock-preview"

	// https://developer.github.com/changes/2016-05-12-reactions-api-preview/
	mediaTypeReactionsPreview = "application/vnd.github.squirrel-girl-preview"

	// https://developer.github.com/changes/2016-04-04-git-signing-api-preview/
	mediaTypeGitSigningPreview = "application/vnd.github.cryptographer-preview+json"

	// https://developer.github.com/changes/2016-05-23-timeline-preview-api/
	mediaTypeTimelinePreview = "application/vnd.github.mockingbird-preview+json"

	// https://developer.github.com/changes/2016-06-14-repository-invitations/
	mediaTypeRepositoryInvitationsPreview = "application/vnd.github.swamp-thing-preview+json"

	// https://developer.github.com/changes/2016-07-06-github-pages-preiew-api/
	mediaTypePagesPreview = "application/vnd.github.mister-fantastic-preview+json"

	// https://developer.github.com/changes/2016-09-14-projects-api/
	mediaTypeProjectsPreview = "application/vnd.github.inertia-preview+json"

	// https://developer.github.com/changes/2016-09-14-Integrations-Early-Access/
	mediaTypeIntegrationPreview = "application/vnd.github.machine-man-preview+json"

	// https://developer.github.com/changes/2017-01-05-commit-search-api/
	mediaTypeCommitSearchPreview = "application/vnd.github.cloak-preview+json"

	// https://developer.github.com/changes/2017-02-28-user-blocking-apis-and-webhook/
	mediaTypeBlockUsersPreview = "application/vnd.github.giant-sentry-fist-preview+json"

	// https://developer.github.com/changes/2017-02-09-community-health/
	mediaTypeRepositoryCommunityHealthMetricsPreview = "application/vnd.github.black-panther-preview+json"

	// https://developer.github.com/changes/2017-05-23-coc-api/
	mediaTypeCodesOfConductPreview = "application/vnd.github.scarlet-witch-preview+json"

	// https://developer.github.com/changes/2017-07-17-update-topics-on-repositories/
	mediaTypeTopicsPreview = "application/vnd.github.mercy-preview+json"

	// https://developer.github.com/changes/2017-08-30-preview-nested-teams/
	mediaTypeNestedTeamsPreview = "application/vnd.github.hellcat-preview+json"

	// https://developer.github.com/changes/2017-11-09-repository-transfer-api-preview/
	mediaTypeRepositoryTransferPreview = "application/vnd.github.nightshade-preview+json"

	// https://developer.github.com/changes/2018-01-25-organization-invitation-api-preview/
	mediaTypeOrganizationInvitationPreview = "application/vnd.github.dazzler-preview+json"

	// https://developer.github.com/changes/2018-03-16-protected-branches-required-approving-reviews/
	mediaTypeRequiredApprovingReviewsPreview = "application/vnd.github.luke-cage-preview+json"

	// https://developer.github.com/changes/2018-02-22-label-description-search-preview/
	mediaTypeLabelDescriptionSearchPreview = "application/vnd.github.symmetra-preview+json"

	// https://developer.github.com/changes/2018-02-07-team-discussions-api/
	mediaTypeTeamDiscussionsPreview = "application/vnd.github.echo-preview+json"

	// https://developer.github.com/changes/2018-03-21-hovercard-api-preview/
	mediaTypeHovercardPreview = "application/vnd.github.hagar-preview+json"

	// https://developer.github.com/changes/2018-01-10-lock-reason-api-preview/
	mediaTypeLockReasonPreview = "application/vnd.github.sailor-v-preview+json"

	// https://developer.github.com/changes/2018-05-07-new-checks-api-public-beta/
	mediaTypeCheckRunsPreview = "application/vnd.github.antiope-preview+json"

	// https://developer.github.com/enterprise/2.13/v3/repos/pre_receive_hooks/
	mediaTypePreReceiveHooksPreview = "application/vnd.github.eye-scream-preview"
)

// A Client manages communication with the GitHub API.
type Client struct {
	clientMu sync.Mutex   // clientMu protects the client during calls that modify the CheckRedirect func.
	client   *http.Client // HTTP client used to communicate with the API.

	// Base URL for API requests. Defaults to the public GitHub API, but can be
	// set to a domain endpoint to use with GitHub Enterprise. BaseURL should
	// always be specified with a trailing slash.
	BaseURL *url.URL

	// Base URL for uploading files.
	UploadURL *url.URL

	// User agent used when communicating with the GitHub API.
	UserAgent string

	rateMu     sync.Mutex
	rateLimits [categories]Rate // Rate limits for the client as determined by the most recent API calls.

	common service // Reuse a single struct instead of allocating one for each service on the heap.

	// Services used for talking to different parts of the GitHub API.
	Activity       *ActivityService
	Admin          *AdminService
	Apps           *AppsService
	Authorizations *AuthorizationsService
	Checks         *ChecksService
	Gists          *GistsService
	Git            *GitService
	Gitignores     *GitignoresService
	Issues         *IssuesService
	Licenses       *LicensesService
	Marketplace    *MarketplaceService
	Migrations     *MigrationService
	Organizations  *OrganizationsService
	Projects       *ProjectsService
	PullRequests   *PullRequestsService
	Reactions      *ReactionsService
	Repositories   *RepositoriesService
	Search         *SearchService
	Teams          *TeamsService
	Users          *UsersService
}

type service struct {
	client *Client
}

// ListOptions specifies the optional parameters to various List methods that
// support pagination.
type ListOptions struct {
	// For paginated result sets, page of results to retrieve.
	Page int `url:"page,omitempty"`

	// For paginated result sets, the number of results to include per page.
	PerPage int `url:"per_page,omitempty"`
}

// UploadOptions specifies the parameters to methods that support uploads.
type UploadOptions struct {
	Name string `url:"name,omitempty"`
}

// RawType represents type of raw format of a request instead of JSON.
type RawType uint8

const (
	// Diff format.
	Diff RawType = 1 + iota
	// Patch format.
	Patch
)

// RawOptions specifies parameters when user wants to get raw format of
// a response instead of JSON.
type RawOptions struct {
	Type RawType
}

// addOptions adds the parameters in opt as URL query parameters to s. opt
// must be a struct whose fields may contain "url" tags.
func addOptions(s string, opt interface{}) (string, error) {
	v := reflect.ValueOf(opt)
	if v.Kind() == reflect.Ptr && v.IsNil() {
		return s, nil
	}

	u, err := url.Parse(s)
	if err != nil {
		return s, err
	}

	qs, err := query.Values(opt)
	if err != nil {
		return s, err
	}

	u.RawQuery = qs.Encode()
	return u.String(), nil
}

// NewClient returns a new GitHub API client. If a nil httpClient is
// provided, http.DefaultClient will be used. To use API methods which require
// authentication, provide an http.Client that will perform the authentication
// for you (such as that provided by the golang.org/x/oauth2 library).
func NewClient(httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	baseURL, _ := url.Parse(defaultBaseURL)
	uploadURL, _ := url.Parse(uploadBaseURL)

	c := &Client{client: httpClient, BaseURL: baseURL, UserAgent: userAgent, UploadURL: uploadURL}
	c.common.client = c
	c.Activity = (*ActivityService)(&c.common)
	c.Admin = (*AdminService)(&c.common)
	c.Apps = (*AppsService)(&c.common)
	c.Authorizations = (*AuthorizationsService)(&c.common)
	c.Checks = (*ChecksService)(&c.common)
	c.Gists = (*GistsService)(&c.common)
	c.Git = (*GitService)(&c.common)
	c.Gitignores = (*GitignoresService)(&c.common)
	c.Issues = (*IssuesService)(&c.common)
	c.Licenses = (*LicensesService)(&c.common)
	c.Marketplace = &MarketplaceService{client: c}
	c.Migrations = (*MigrationService)(&c.common)
	c.Organizations = (*OrganizationsService)(&c.common)
	c.Projects = (*ProjectsService)(&c.common)
	c.PullRequests = (*PullRequestsService)(&c.common)
	c.Reactions = (*ReactionsService)(&c.common)
	c.Repositories = (*RepositoriesService)(&c.common)
	c.Search = (*SearchService)(&c.common)
	c.Teams = (*TeamsService)(&c.common)
	c.Users = (*UsersService)(&c.common)
	return c
}

// NewEnterpriseClient returns a new GitHub API client with provided
// base URL and upload URL (often the same URL).
// If either URL does not have a trailing slash, one is added automatically.
// If a nil httpClient is provided, http.DefaultClient will be used.
//
// Note that NewEnterpriseClient is a convenience helper only;
// its behavior is equivalent to using NewClient, followed by setting
// the BaseURL and UploadURL fields.
func NewEnterpriseClient(baseURL, uploadURL string, httpClient *http.Client) (*Client, error) {
	baseEndpoint, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}
	if !strings.HasSuffix(baseEndpoint.Path, "/") {
		baseEndpoint.Path += "/"
	}

	uploadEndpoint, err := url.Parse(uploadURL)
	if err != nil {
		return nil, err
	}
	if !strings.HasSuffix(uploadEndpoint.Path, "/") {
		uploadEndpoint.Path += "/"
	}

	c := NewClient(httpClient)
	c.BaseURL = baseEndpoint
	c.UploadURL = uploadEndpoint
	return c, nil
}

// NewRequest creates an API request. A relative URL can be provided in urlStr,
// in which case it is resolved relative to the BaseURL of the Client.
// Relative URLs should always be specified without a preceding slash. If
// specified, the value pointed to by body is JSON encoded and included as the
// request body.
func (c *Client) NewRequest(method, urlStr string, body interface{}) (*http.Request, error) {
	if !strings.HasSuffix(c.BaseURL.Path, "/") {
		return nil, fmt.Errorf("BaseURL must have a trailing slash, but %q does not", c.BaseURL)
	}
	u, err := c.BaseURL.Parse(urlStr)
	if err != nil {
		return nil, err
	}

	var buf io.ReadWriter
	if body != nil {
		buf = new(bytes.Buffer)
		enc := json.NewEncoder(buf)
		enc.SetEscapeHTML(false)
		err := enc.Encode(body)
		if err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequest(method, u.String(), buf)
	if err != nil {
		return nil, err
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", mediaTypeV3)
	if c.UserAgent != "" {
		req.Header.Set("User-Agent", c.UserAgent)
	}
	return req, nil
}

// NewUploadRequest creates an upload request. A relative URL can be provided in
// urlStr, in which case it is resolved relative to the UploadURL of the Client.
// Relative URLs should always be specified without a preceding slash.
func (c *Client) NewUploadRequest(urlStr string, reader io.Reader, size int64, mediaType string) (*http.Request, error) {
	if !strings.HasSuffix(c.UploadURL.Path, "/") {
		return nil, fmt.Errorf("UploadURL must have a trailing slash, but %q does not", c.UploadURL)
	}
	u, err := c.UploadURL.Parse(urlStr)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", u.String(), reader)
	if err != nil {
		return nil, err
	}
	req.ContentLength = size

	if mediaType == "" {
		mediaType = defaultMediaType
	}
	req.Header.Set("Content-Type", mediaType)
	req.Header.Set("Accept", mediaTypeV3)
	req.Header.Set("User-Agent", c.UserAgent)
	return req, nil
}

// Response is a GitHub API response. This wraps the standard http.Response
// returned from GitHub and provides convenient access to things like
// pagination links.
type Response struct {
	*http.Response

	// These fields provide the page values for paginating through a set of
	// results. Any or all of these may be set to the zero value for
	// responses that are not part of a paginated set, or for which there
	// are no additional pages.

	NextPage  int
	PrevPage  int
	FirstPage int
	LastPage  int

	Rate
}

// newResponse creates a new Response 