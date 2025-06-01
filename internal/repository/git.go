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

func Diff(ctx context.Context, repo *db.Repo, baseRef, changeRef string) (*object.Patch, error) {
	r, err := openGitRepo(ctx, repo)
	if err != nil {
		return nil, err
	}
	auth := repo.Auth()

	baseHash, err := ensureRevision(ctx, r, baseRef, auth)
	if err != nil {
		return nil, fmt.Errorf("base %s: %w", baseRef, err)
	}
	changeHash, err := ensureRevision(ctx, r, changeRef, auth)
	if err != nil {
		return nil, fmt.Errorf("change %s: %w", changeRef, err)
	}

	baseCommit, err := r.CommitObject(baseHash)
	if err != nil {
		return nil, fmt.Errorf("load base %s: %w", baseHash, err)
	}
	changeCommit, err := r.CommitObject(changeHash)
	if err != nil {
		return nil, fmt.Errorf("load change %s: %w", changeHash, err)
	}

	patch, err := baseCommit.PatchContext(ctx, changeCommit)
	if err != nil {
		return nil, fmt.Errorf("diff %s..%s: %w", baseRef, changeRef, err)
	}
	return patch, nil
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

func openGitRepo(ctx context.Context, repo *db.Repo) (*git.Repository, error) {
	repoPath := filepath.Join(tempDir, repo.Name)
	if err := ensureGitRepoExists(ctx, repo); err != nil {
		return nil, err
	}
	return git.PlainOpen(repoPath)
}

func ensureGitRepoExists(ctx context.Context, repo *db.Repo) error {
	repoPath := filepath.Join(tempDir, repo.Name)
	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		if err := os.MkdirAll(repoPath, 0777); err != nil {
			return errors.Join(fmt.Errorf("failed to create directory %s", repoPath), err)
		}
		if err := cloneGitRepo(ctx, repo); err != nil {
			return errors.Join(fmt.Errorf("failed to clone git repository %s", repo.Url), err)
		}
		return nil
	} else {
		return nil
	}
}

func cloneGitRepo(ctx context.Context, repo *db.Repo) error {
	repoPath := filepath.Join(tempDir, repo.Name)
	_, err := git.PlainCloneContext(ctx, repoPath, false, &git.CloneOptions{
		URL:  repo.Url,
		Auth: repo.Auth(),
	})
	if err != nil {
		return errors.Join(fmt.Errorf("failed to clone git repository %s into %s", repo.Url, repoPath), err)
	}
	return nil
}
