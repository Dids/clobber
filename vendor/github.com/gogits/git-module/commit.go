// Copyright 2015 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package git

import (
	"bufio"
	"bytes"
	"container/list"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/mcuadros/go-version"
)

// Commit represents a git commit.
type Commit struct {
	Tree
	ID            sha1 // The ID of this commit object
	Author        *Signature
	Committer     *Signature
	CommitMessage string

	parents        []sha1 // SHA1 strings
	submoduleCache *objectCache
}

// Message returns the commit message. Same as retrieving CommitMessage directly.
func (c *Commit) Message() string {
	return c.CommitMessage
}

// Summary returns first line of commit message.
func (c *Commit) Summary() string {
	return strings.Split(c.CommitMessage, "\n")[0]
}

// ParentID returns oid of n-th parent (0-based index).
// It returns nil if no such parent exists.
func (c *Commit) ParentID(n int) (sha1, error) {
	if n >= len(c.parents) {
		return sha1{}, ErrNotExist{"", ""}
	}
	return c.parents[n], nil
}

// Parent returns n-th parent (0-based index) of the commit.
func (c *Commit) Parent(n int) (*Commit, error) {
	id, err := c.ParentID(n)
	if err != nil {
		return nil, err
	}
	parent, err := c.repo.getCommit(id)
	if err != nil {
		return nil, err
	}
	return parent, nil
}

// ParentCount returns number of parents of the commit.
// 0 if this is the root commit,  otherwise 1,2, etc.
func (c *Commit) ParentCount() int {
	return len(c.parents)
}

func isImageFile(data []byte) (string, bool) {
	contentType := http.DetectContentType(data)
	if strings.Index(contentType, "image/") != -1 {
		return contentType, true
	}
	return contentType, false
}

func (c *Commit) IsImageFile(name string) bool {
	blob, err := c.GetBlobByPath(name)
	if err != nil {
		return false
	}

	dataRc, err := blob.Data()
	if err != nil {
		return false
	}
	buf := make([]byte, 1024)
	n, _ := dataRc.Read(buf)
	buf = buf[:n]
	_, isImage := isImageFile(buf)
	return isImage
}

// GetCommitByPath return the commit of relative path object.
func (c *Commit) GetCommitByPath(relpath string) (*Commit, error) {
	return c.repo.getCommitByPathWithID(c.ID, relpath)
}

// AddAllChanges marks local changes to be ready for commit.
func AddChanges(repoPath string, all bool, files ...string) error {
	cmd := NewCommand("add")
	if all {
		cmd.AddArguments("--all")
	}
	_, err := cmd.AddArguments(files...).RunInDir(repoPath)
	return err
}

type CommitChangesOptions struct {
	Committer *Signature
	Author    *Signature
	Message   string
}

// CommitChanges commits local changes with given committer, author and message.
// If author is nil, it will be the same as committer.
func CommitChanges(repoPath string, opts CommitChangesOptions) error {
	cmd := NewCommand()
	if opts.Committer != nil {
		cmd.AddEnvs("GIT_COMMITTER_NAME="+opts.Committer.Name, "GIT_COMMITTER_EMAIL="+opts.Committer.Email)
	}
	cmd.AddArguments("commit")

	if opts.Author == nil {
		opts.Author = opts.Committer
	}
	if opts.Author != nil {
		cmd.AddArguments(fmt.Sprintf("--author='%s <%s>'", opts.Author.Name, opts.Author.Email))
	}
	cmd.AddArguments("-m", opts.Message)

	_, err := cmd.RunInDir(repoPath)
	// No stderr but exit status 1 means nothing to commit.
	if err != nil && err.Error() == "exit status 1" {
		return nil
	}
	return err
}

func commitsCount(repoPath, revision, relpath string) (int64, error) {
	var cmd *Command
	isFallback := false
	if version.Compare(gitVersion, "1.8.0", "<") {
		isFallback = true
		cmd = NewCommand("log", "--pretty=format:''")
	} else {
		cmd = NewCommand("rev-list", "--count")
	}
	cmd.AddArguments(revision)
	if len(relpath) > 0 {
		cmd.AddArguments("--", relpath)
	}

	stdout, err := cmd.RunInDir(repoPath)
	if err != nil {
		return 0, err
	}

	if isFallback {
		return int64(strings.Count(stdout, "\n")) + 1, nil
	}
	return strconv.ParseInt(strings.TrimSpace(stdout), 10, 64)
}

func (c *Commit) CommitsCount() (int64, error) {
	return CommitsCount(c.repo.Path, c.ID.String())
}

func (c *Commit) CommitsByRangeSize(page, size int) (*list.List, error) {
	return c.repo.CommitsByRangeSize(c.ID.String(), page, size)
}

func (c *Commit) CommitsByRange(page int) (*list.List, error) {
	return c.repo.CommitsByRange(c.ID.String(), page)
}

func (c *Commit) CommitsBefore() (*list.List, error) {
	return c.repo.getCommitsBefore(c.ID)
}

func (c *Commit) CommitsBeforeLimit(num int) (*list.List, error) {
	return c.repo.getCommitsBeforeLimit(c.ID, num)
}

func (c *Commit) CommitsBeforeUntil(commitID string) (*list.List, error) {
	endCommit, err := c.repo.GetCommit(commitID)
	if err != nil {
		return nil, err
	}
	return c.repo.CommitsBetween(c, endCommit)
}

func (c *Commit) SearchCommits(keyword string) (*list.List, error) {
	return c.repo.searchCommits(c.ID, keyword)
}

func (c *Commit) GetFilesChangedSinceCommit(pastCommit string) ([]string, error) {
	return c.repo.getFilesChanged(pastCommit, c.ID.String())
}

func (c *Commit) GetSubModules() (*objectCache, error) {
	if c.submoduleCache != nil {
		return c.submoduleCache, nil
	}

	entry, err := c.GetTreeEntryByPath(".gitmodules")
	if err != nil {
		return nil, err
	}
	rd, err := entry.Blob().Data()
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(rd)
	c.submoduleCache = newObjectCache()
	var ismodule bool
	var path string
	for scanner.Scan() {
		if strings.HasPrefix(scanner.Text(), "[submodule") {
			ismodule = true
			continue
		}
		if ismodule {
			fields := strings.Split(scanner.Text(), "=")
			k := strings.TrimSpace(fields[0])
			if k == "path" {
				path = strings.TrimSpace(fields[1])
			} else if k == "url" {
				c.submoduleCache.Set(path, &SubModule{path, strings.TrimSpace(fields[1])})
				ismodule = false
			}
		}
	}

	return c.submoduleCache, nil
}

func (c *Commit) GetSubModule(entryname string) (*SubModule, error) {
	modules, err := c.GetSubModules()
	if err != nil {
		return nil, err
	}

	module, has := modules.Get(entryname)
	if has {
		return module.(*SubModule), nil
	}
	return nil, nil
}

// CommitFileStatus represents status of files in a commit.
type CommitFileStatus struct {
	Added    []string
	Removed  []string
	Modified []string
}

func NewCommitFileStatus() *CommitFileStatus {
	return &CommitFileStatus{
		[]string{}, []string{}, []string{},
	}
}

// GetCommitFileStatus returns file status of commit in given repository.
func GetCommitFileStatus(repoPath, commitID string) (*CommitFileStatus, error) {
	stdout, w := io.Pipe()
	done := make(chan struct{})
	fileStatus := NewCommitFileStatus()
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			fields := strings.Fields(scanner.Text())
			if len(fields) < 2 {
				continue
			}

			switch fields[0][0] {
			case 'A':
				fileStatus.Added = append(fileStatus.Added, fields[1])
			case 'D':
				fileStatus.Removed = append(fileStatus.Removed, fields[1])
			case 'M':
				fileStatus.Modified = append(fileStatus.Modified, fields[1])
			}
		}
		done <- struct{}{}
	}()

	stderr := new(bytes.Buffer)
	err := NewCommand("show", "--name-status", "--pretty=format:''", commitID).RunInDirPipeline(repoPath, w, stderr)
	w.Close() // Close writer to exit parsing goroutine
	if err != nil {
		return nil, concatenateError(err, stderr.String())
	}

	<-done
	return fileStatus, nil
}

// FileStatus returns file status of commit.
func (c *Commit) FileStatus() (*CommitFileStatus, error) {
	return GetCommitFileStatus(c.repo.Path, c.ID.String())
}

// GetFullCommitID returns full length (40) of commit ID by given short SHA in a repository.
func GetFullCommitID(repoPath, shortID string) (string, error) {
	if len(shortID) >= 40 {
		return shortID, nil
	}

	commitID, err := NewCommand("rev-parse", shortID).RunInDir(repoPath)
	if err != nil {
		if strings.Contains(err.Error(), "exit status 128") {
			return "", ErrNotExist{shortID, ""}
		}
		return "", err
	}
	return strings.TrimSpace(commitID), nil
}
