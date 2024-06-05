// SPDX-FileCopyrightText: Copyright © 2020-2023 Serpent OS Developers
//
// SPDX-License-Identifier: MPL-2.0

package push

import (
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"

	"github.com/GZGavinZhao/autobuild/common"
	"github.com/go-git/go-git/v5"
	// "github.com/go-git/go-git/v5/config"
	// "github.com/go-git/go-git/v5/plumbing/transport/ssh"
)

const (
	user = "build-controller"
	host = "build.getsol.us"
)

func Publish(pkg common.Package, prePush bool) (job Job, err error) {
	root := pkg.Root
	relp, err := filepath.Rel(root, pkg.Path)
	if err != nil {
		err = fmt.Errorf("push.Publish: unable to convert %s to relative path to %s: %w", pkg.Path, pkg.Root, err)
		return
	}

	// Open the repository
	repo, err := git.PlainOpen(root)
	if err != nil {
		err = fmt.Errorf("Failed to open git repository at %s: %w", root, err)
		return
	}

	ref, err := repo.Head()
	if err != nil {
		err = fmt.Errorf("Failed to get HEAD of repository at %s: %w", root, err)
		return
	}

	if ref.Name().String() != "refs/heads/main" {
		err = errors.New("push.Publish: not on main branch!")
		return
	}

	// br, err := repo.Branch(ref.Name().Short())
	// if err != nil {
	// 	err = fmt.Errorf("push.Publish: failed to get branch of HEAD: %w", err)
	// 	return
	// }

	// rm, err := repo.Remote(br.Remote)
	// if err != nil {
	// 	err = fmt.Errorf("push.Publish: failed to get remote of branch %s: %w", br.Name, err)
	// 	return
	// }

	// localRef := ref.Name().String()
	// remoteRef := fmt.Sprintf("refs/remotes/%s/%s", rm.Config().Name, br.Name)
	// refspec := config.RefSpec(fmt.Sprintf("+%s:%s", localRef, remoteRef))
	// if err = refspec.Validate(); err != nil {
	// 	err = fmt.Errorf("push.Publish: failed to validate refspec %s: %w", refspec.String(), err)
	// 	return
	// }

	// sshKeyPath, ok := os.LookupEnv("AUTOBUILD_SSHKEY")
	// var auth *ssh.PublicKeys
	// if ok {
	// 	auth, err = ssh.NewPublicKeysFromFile("git", sshKeyPath, "")
	// 	if err != nil {
	// 		err = fmt.Errorf("push.Publish: failed to obtain ssh private key from %s: %w", sshKeyPath, err)
	// 		return
	// 	}
	// 	println("Picked up key", sshKeyPath, auth.String())
	// }

	// println("Pushing to remote", rm.Config().Name, rm.Config().URLs)
	// err = rm.Push(&git.PushOptions{
	// 	RemoteName: rm.Config().Name,
	// 	RefSpecs:   []config.RefSpec{refspec},
	// 	Auth:       auth,
	// })
	// if err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
	// 	err = fmt.Errorf("push.Publish: failed to push to remote %s: %w", rm.Config().Name, err)
	// 	return
	// }

	// go-git cannot pick up the correct user/SSH public key to push! Bruh!
	var output []byte
	if prePush {
		pushCmd := exec.Command("git", "push")
		pushCmd.Dir = pkg.Root
		if output, err = pushCmd.CombinedOutput(); err != nil {
			err = fmt.Errorf("push.Publish: failed to push to remote: %w, stderr: %s", err, string(output))
			return
		}
	}

	args := []string{
		fmt.Sprintf("%s@%s", user, host),
		"build",
		pkg.Name,
		fmt.Sprintf("%s-%s-%d", pkg.Name, pkg.Version, pkg.Release),
		relp,
		ref.Hash().String(),
		"YnkgYXV0b2J1aWxk", // "by autobuild"
	}
	cmd := exec.Command("ssh", args...)
	if output, err = cmd.Output(); err != nil {
		err = fmt.Errorf("push.Publish: failed to publish package %s using args %q: %w", pkg.Name, args, err)
		return
	}

	if err = json.Unmarshal(output, &job); err != nil {
		err = fmt.Errorf("push.Publish: failed to unmarshall json output to job: %w", err)
		return
	}

	return
}

func Query(jobid int) (job Job, err error) {
	args := []string{
		fmt.Sprintf("%s@%s", user, host),
		"query",
		fmt.Sprint(jobid),
	}
	cmd := exec.Command("ssh", args...)
	output, err := cmd.Output()
	if err != nil {
		err = fmt.Errorf("Failed to query job %d using args %q: %w", jobid, args, err)
		return
	}

	err = json.Unmarshal(output, &job)
	if err != nil {
		err = fmt.Errorf("Failed to unmarshall json output to job: %w", err)
		return
	}

	return
}
