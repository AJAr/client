// Copyright 2018 Keybase, Inc. All rights reserved. Use of
// this source code is governed by the included BSD license.

package client

import (
	"errors"
	"fmt"
	"os"
	"regexp"

	"github.com/araddon/dateparse"
	"github.com/keybase/cli"
	"github.com/keybase/client/go/chat/search"
	"github.com/keybase/client/go/libcmdline"
	"github.com/keybase/client/go/libkb"
	"github.com/keybase/client/go/protocol/chat1"
	"github.com/keybase/client/go/protocol/gregor1"
	keybase1 "github.com/keybase/client/go/protocol/keybase1"
	"github.com/keybase/go-framed-msgpack-rpc/rpc"
	isatty "github.com/mattn/go-isatty"
	"golang.org/x/net/context"
)

type CmdChatSearch struct {
	libkb.Contextified
	resolvingRequest chatConversationResolvingRequest
	query            string
	opts             chat1.SearchOpts
	isRegex          bool
	hasTTY           bool
}

func NewCmdChatSearchRunner(g *libkb.GlobalContext) *CmdChatSearch {
	return &CmdChatSearch{
		Contextified: libkb.NewContextified(g),
	}
}

func newCmdChatSearch(cl *libcmdline.CommandLine, g *libkb.GlobalContext) cli.Command {
	return cli.Command{
		Name:         "search",
		Usage:        "Search via regex within a conversation",
		ArgumentHelp: "<conversation> <query>",
		Action: func(c *cli.Context) {
			cl.ChooseCommand(NewCmdChatSearchRunner(g), "search", c)
			cl.SetNoStandalone()
		},
		Flags: append(getConversationResolverFlags(),
			cli.BoolFlag{
				Name:  "r, regex",
				Usage: "Make the given query a regex",
			},
			cli.IntFlag{
				Name:  "max-hits",
				Value: 10,
				Usage: fmt.Sprintf("Specify the maximum number of search hits to get. Maximum value is %d.", search.MaxAllowedSearchHits),
			},
			cli.StringFlag{
				Name:  "sent-by",
				Value: "",
				Usage: "Filter search results by the username of the sender.",
			},
			cli.StringFlag{
				Name:  "sent-before",
				Value: "",
				Usage: "Filter search results by the message creation time. Mutually exclusive with sent-after.",
			},
			cli.StringFlag{
				Name:  "sent-after",
				Value: "",
				Usage: "Filter search results by the message creation time. Mutually exclusive with sent-before.",
			},
			cli.IntFlag{
				Name:  "max-messages",
				Value: 10000,
				Usage: fmt.Sprintf("Specify the maximum number of messages to search. Maximum value is %d.", search.MaxAllowedSearchMessages),
			},
			cli.IntFlag{
				Name:  "B, before-context",
				Value: 0,
				Usage: "Print number messages of leading context before each match.",
			},
			cli.IntFlag{
				Name:  "A, after-context",
				Value: 0,
				Usage: "Print number of messages of trailing context after each match.",
			},
			cli.IntFlag{
				Name:  "C, context",
				Value: 2,
				Usage: "Print number of messages of leading and trailing context surrounding each match.",
			},
		),
	}
}

func (c *CmdChatSearch) Run() (err error) {
	ui := &ChatUI{
		Contextified: libkb.NewContextified(c.G()),
		terminal:     c.G().UI.GetTerminalUI(),
	}
	protocols := []rpc.Protocol{
		chat1.ChatUiProtocol(ui),
	}
	if err := RegisterProtocolsWithContext(protocols, c.G()); err != nil {
		return err
	}

	if c.resolvingRequest.TlfName != "" {
		if err = annotateResolvingRequest(c.G(), &c.resolvingRequest); err != nil {
			return err
		}
	}
	// TODO: Right now this command cannot be run in standalone at
	// all, even though team chats should work, but there is a bug
	// in finding existing conversations.
	if c.G().Standalone {
		switch c.resolvingRequest.MembersType {
		case chat1.ConversationMembersType_TEAM, chat1.ConversationMembersType_IMPTEAMNATIVE,
			chat1.ConversationMembersType_IMPTEAMUPGRADE:
			c.G().StartStandaloneChat()
		default:
			err = CantRunInStandaloneError{}
			return err
		}
	}

	resolver, err := newChatConversationResolver(c.G())
	if err != nil {
		return err
	}

	ctx := context.TODO()
	conversation, _, err := resolver.Resolve(ctx, c.resolvingRequest, chatConversationResolvingBehavior{
		CreateIfNotExists: false,
		MustNotExist:      false,
		Interactive:       c.hasTTY,
		IdentifyBehavior:  keybase1.TLFIdentifyBehavior_CHAT_CLI,
	})
	if err != nil {
		return err
	}
	conversationInfo := conversation.Info

	arg := chat1.GetSearchRegexpArg{
		ConvID:           conversationInfo.Id,
		IdentifyBehavior: keybase1.TLFIdentifyBehavior_CHAT_CLI,
		Query:            c.query,
		IsRegex:          c.isRegex,
		Opts:             c.opts,
	}

	_, err = resolver.ChatClient.GetSearchRegexp(ctx, arg)
	return err
}

func (c *CmdChatSearch) ParseArgv(ctx *cli.Context) (err error) {
	if len(ctx.Args()) != 2 {
		return errors.New("usage: keybase chat search <conversation> <query>")
	}
	// Get the TLF name from the first position arg
	tlfName := ctx.Args().Get(0)
	if c.resolvingRequest, err = parseConversationResolvingRequest(ctx, tlfName); err != nil {
		return err
	}
	c.query = ctx.Args().Get(1)
	c.opts.SentBy = ctx.String("sent-by")
	sentBeforeStr := ctx.String("sent-before")
	sentAfterStr := ctx.String("sent-after")
	if sentBeforeStr != "" && sentAfterStr != "" {
		return fmt.Errorf("Only one of sent-before and sent-after can be specified")
	}
	if sentBeforeStr != "" {
		sentBefore, err := dateparse.ParseAny(sentBeforeStr)
		if err != nil {
			return err
		}
		c.opts.SentBefore = gregor1.ToTime(sentBefore)
	}
	if sentAfterStr != "" {
		sentAfter, err := dateparse.ParseAny(sentAfterStr)
		if err != nil {
			return err
		}
		c.opts.SentAfter = gregor1.ToTime(sentAfter)
	}

	c.opts.MaxHits = ctx.Int("max-hits")
	if c.opts.MaxHits > search.MaxAllowedSearchHits {
		return fmt.Errorf("max-hits cannot exceed %d.", search.MaxAllowedSearchHits)
	}
	c.opts.MaxMessages = ctx.Int("max-messages")
	if c.opts.MaxMessages > search.MaxAllowedSearchMessages {
		return fmt.Errorf("max-messages cannot exceed %d.", search.MaxAllowedSearchMessages)
	}

	c.opts.AfterContext = ctx.Int("after-context")
	c.opts.BeforeContext = ctx.Int("before-context")
	if c.opts.AfterContext == 0 && c.opts.BeforeContext == 0 {
		context := ctx.Int("context")
		c.opts.BeforeContext = context
		c.opts.AfterContext = context
	}

	c.isRegex = ctx.Bool("regex")
	query := c.query
	if !c.isRegex {
		query = regexp.QuoteMeta(c.query)
	}
	if _, err := regexp.Compile(query); err != nil {
		return err
	}
	c.hasTTY = isatty.IsTerminal(os.Stdin.Fd())

	return nil
}

func (c *CmdChatSearch) GetUsage() libkb.Usage {
	return libkb.Usage{
		Config: true,
		API:    true,
	}
}
