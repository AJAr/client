// Copyright 2015 Keybase, Inc. All rights reserved. Use of
// this source code is governed by the included BSD license.

package client

import (
	"errors"

	"golang.org/x/net/context"

	"github.com/keybase/cli"
	"github.com/keybase/client/go/libcmdline"
	"github.com/keybase/client/go/libkb"
	keybase1 "github.com/keybase/client/go/protocol/keybase1"
)

// CmdSimpleFSSymlink is the 'fs ln' command.
type CmdSimpleFSSymlink struct {
	libkb.Contextified
	src  keybase1.Path
	dest keybase1.Path
}

// NewCmdSimpleFSSymlink creates a new cli.Command.
func NewCmdSimpleFSSymlink(cl *libcmdline.CommandLine, g *libkb.GlobalContext) cli.Command {
	return cli.Command{
		Name:         "ln",
		ArgumentHelp: "<target> <link>",
		Usage:        "create a symlink from link to target",
		Action: func(c *cli.Context) {
			cl.ChooseCommand(&CmdSimpleFSSymlink{Contextified: libkb.NewContextified(g)}, "ln", c)
			cl.SetNoStandalone()
		},
	}
}

// Run runs the command in client/server mode.
func (c *CmdSimpleFSSymlink) Run() error {
	cli, err := GetSimpleFSClient(c.G())
	if err != nil {
		return err
	}

	ctx := context.TODO()
	arg := keybase1.SimpleFSSymlinkArg{
		Src:  c.src,
		Dest: c.dest,
	}

	return cli.SimpleFSSymlink(ctx, arg)
}

// ParseArgv gets the required path argument for this command.
func (c *CmdSimpleFSSymlink) ParseArgv(ctx *cli.Context) error {
	nargs := len(ctx.Args())
	if nargs != 2 {
		return errors.New("ln requires exactly 2 arguments")
	}

	var err error
	var sources []keybase1.Path
	sources, c.dest, err = parseSrcDestArgs(c.G(), ctx, "ln")
	c.src = sources[0] // only one source, checked above
	return err
}

// GetUsage says what this command needs to operate.
func (c *CmdSimpleFSSymlink) GetUsage() libkb.Usage {
	return libkb.Usage{
		Config:    true,
		KbKeyring: true,
		API:       true,
	}
}
