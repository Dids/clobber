// Copyright 2015 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package git

import (
	"fmt"
	"strings"

	"github.com/mcuadros/go-version"
)

const BRANCH_PREFIX = "refs/heads/"

// IsReferenceExist returns true if given reference exists in the repository.
func IsReferenceExist(repoPath, name string) bool {
	_, err := NewCommand("show-ref", "--verify", name).RunInDir(repoPath)
	return err == nil
}

// IsBranchExist returns true if given branch exists in the repository.
func IsBranchExist(repoPath, name string) bool {
	return IsReferenceExist(repoPath, BRANCH_PREFIX+name)
}

func (repo *Repository) IsBranchExist(name string) bool {
	return IsBranchExist(repo.Path, name)
}

// Branch represents a Git branch.
type Branch struct {
	Name string
	Path string
}

// GetHEADBranch returns corresponding branch of HEAD.
func (repo *Repository) GetHEADBranch() (*Branch, error) {
	stdout, err := NewCommand("symbolic-ref", "HEAD").RunInDir(repo.Path)
	if err != nil {
		return nil, err
	}
	stdout = strings.TrimSpace(stdout)

	if !strings.HasPrefix(stdout, BRANCH_PREFIX) {
		return nil, fmt.Errorf("invalid HEAD branch: %v", stdout)
	}

	return &Branch{
		Name: stdout[len(BRANCH_PREFIX):],
		Path: stdout,
	}, nil
}

// SetDefaultBranch sets default branch of repository.
func (repo *Repository) SetDefaultBranch(name string) error {
	if version.Compare(gitVersion, "1.7.10", "<") {
		return ErrUnsupportedVersion{"1.7.10"}
	}

	_, err := NewCommand("symbolic-ref", "HEAD", BRANCH_PREFIX+name).RunInDir(repo.Path)
	return err
}

// GetBranches returns all branches of the repository.
func (repo *Repository) GetBranches() ([]string, error) {
	stdout, err := NewCommand("show-ref", "--heads").RunInDir(repo.Path)
	if err != nil {
		return nil, err
	}

	infos := strings.Split(stdout, "\n")
	branches := make([]string, len(infos)-1)
	for i, info := range infos[:len(infos)-1] {
		fields := strings.Fields(info)
		if len(fields) != 2 {
			continue // NOTE: I should believe git will not give me wrong string.
		}
		branches[i] = strings.TrimPrefix(fields[1], BRANCH_PREFIX)
	}
	return branches, nil
}

// Option(s) for delete branch
type DeleteBranchOptions struct {
	Force bool
}

// DeleteBranch deletes a branch from given repository path.
func DeleteBranch(repoPath, name string, opts DeleteBranchOptions) error {
	cmd := NewCommand("branch")

	if opts.Force {
		cmd.AddArguments("-D")
	} else {
		cmd.AddArguments("-d")
	}

	cmd.AddArguments(name)
	_, err := cmd.RunInDir(repoPath)

	return err
}

// DeleteBranch deletes a branch from repository.
func (repo *Repository) DeleteBranch(name string, opts DeleteBranchOptions) error {
	return DeleteBranch(repo.Path, name, opts)
}

// AddRemote adds a new remote to repository.
func (repo *Repository) AddRemote(name, url string, fetch bool) error {
	cmd := NewCommand("remote", "add")
	if fetch {
		cmd.AddArguments("-f")
	}
	cmd.AddArguments(name, url)

	_, err := cmd.RunInDir(repo.Path)
	return err
}

// RemoveRemote removes a remote from repository.
func (repo *Repository) RemoveRemote(name string) error {
	_, err := NewCommand("remote", "remove", name).RunInDir(repo.Path)
	return err
}
