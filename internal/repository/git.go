// ViewRe is a web-based code review tool.
// Copyright (C) 2025  Frank Mayer
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package repository

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"viewre/internal/db"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
)

var mutex = &sync.Mutex{}

func CheckoutCommit(ctx context.Context, repo *db.Repo, commitRev string) (string, error) {
	mutex.Lock()
	defer mutex.Unlock()

	repoPath := filepath.Join(tempDir, repo.Name, commitRev[0:4], commitRev[4:])

	r, err := openGitRepo(ctx, repo, repoPath)
	if err != nil {
		return "", err
	}
	auth := repo.Auth()

	commitHash, err := ensureRevision(ctx, r, commitRev, auth)
	if err != nil {
		return "", fmt.Errorf("pull %s: %w", commitRev, err)
	}

	commitObj, err := r.CommitObject(commitHash)
	if err != nil {
		return "", fmt.Errorf("get commit commit %s: %w", commitHash, err)
	}

	w, err := r.Worktree()
	if err != nil {
		return "", fmt.Errorf("open worktree: %w", err)
	}

	if err := w.Checkout(&git.CheckoutOptions{
		Hash:  commitObj.Hash,
		Force: true,
	}); err != nil {
		return "", fmt.Errorf("checkout %s: %w", commitRev, err)
	}

	return repoPath, nil
}

func Diff(ctx context.Context, repo *db.Repo, baseRef, changeRef string) (string, string, *object.Patch, error) {
	mutex.Lock()
	defer mutex.Unlock()

	repoPath := filepath.Join(tempDir, repo.Name, "HEAD")
	r, err := openGitRepo(ctx, repo, repoPath)
	if err != nil {
		return "", "", nil, err
	}
	auth := repo.Auth()

	baseHash, err := ensureRevision(ctx, r, baseRef, auth)
	if err != nil {
		return "", "", nil, fmt.Errorf("base %s: %w", baseRef, err)
	}
	changeHash, err := ensureRevision(ctx, r, changeRef, auth)
	if err != nil {
		return "", "", nil, fmt.Errorf("change %s: %w", changeRef, err)
	}

	baseCommit, err := r.CommitObject(baseHash)
	if err != nil {
		return "", "", nil, fmt.Errorf("load base %s: %w", baseHash, err)
	}
	changeCommit, err := r.CommitObject(changeHash)
	if err != nil {
		return "", "", nil, fmt.Errorf("load change %s: %w", changeHash, err)
	}

	patch, err := baseCommit.PatchContext(ctx, changeCommit)
	if err != nil {
		return "", "", nil, fmt.Errorf("diff %s..%s: %w", baseRef, changeRef, err)
	}

	return baseCommit.Hash.String(), changeCommit.Hash.String(), patch, nil
}

func ensureRevision(ctx context.Context, r *git.Repository, rev string, auth transport.AuthMethod) (plumbing.Hash, error) {
	h, err := r.ResolveRevision(plumbing.Revision(rev))
	if err == nil {
		return *h, nil
	}
	if ferr := r.FetchContext(ctx, &git.FetchOptions{
		RemoteName: "origin",
		Auth:       auth,
		Force:      true,
		RefSpecs: []config.RefSpec{
			"+refs/heads/*:refs/remotes/origin/*",
			"+refs/tags/*:refs/tags/*",
		},
	}); ferr != nil && ferr != git.NoErrAlreadyUpToDate {
		return plumbing.ZeroHash, fmt.Errorf("fetch failed: %w", ferr)
	}
	h, err = r.ResolveRevision(plumbing.Revision(rev))
	if err != nil {
		return plumbing.ZeroHash, fmt.Errorf("resolve %q: %w", rev, err)
	}
	return *h, nil
}

func openGitRepo(ctx context.Context, repo *db.Repo, repoPath string) (*git.Repository, error) {
	if err := ensureGitRepoExists(ctx, repo, repoPath); err != nil {
		return nil, err
	}
	return git.PlainOpen(repoPath)
}

func ensureGitRepoExists(ctx context.Context, repo *db.Repo, repoPath string) error {
	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		if err := os.MkdirAll(repoPath, 0777); err != nil {
			return errors.Join(fmt.Errorf("failed to create directory %s", repoPath), err)
		}
	}
	dirContents, err := os.ReadDir(repoPath)
	if err != nil {
		return errors.Join(fmt.Errorf("failed to read directory %s", repoPath), err)
	}
	if len(dirContents) == 0 {
		if err := cloneGitRepo(ctx, repo, repoPath); err != nil {
			return errors.Join(fmt.Errorf("failed to clone git repository %s", repo.Url), err)
		}
	}
	return nil
}

func cloneGitRepo(ctx context.Context, repo *db.Repo, repoPath string) error {
	_, err := git.PlainCloneContext(ctx, repoPath, false, &git.CloneOptions{
		URL:  repo.Url,
		Auth: repo.Auth(),
	})
	if err != nil {
		return errors.Join(fmt.Errorf("failed to clone git repository %s into %s", repo.Url, repoPath), err)
	}
	return nil
}
