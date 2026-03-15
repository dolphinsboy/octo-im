package cmd

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/WuKongIM/WuKongIM/pkg/wkdb/v2"
	wkinspect "github.com/WuKongIM/WuKongIM/pkg/wkdb/v3/inspect"
	v3key "github.com/WuKongIM/WuKongIM/pkg/wkdb/v3/key"
	"github.com/spf13/cobra"
)

type dbCMD struct {
	ctx *WuKongIMContext
}

type dbCommandOptions struct {
	dataDir   string
	format    string
	limit     int
	slotCount int
}

type dbTable struct {
	headers []string
	rows    [][]string
}

type uidFlags struct {
	uid string
}

type channelFlags struct {
	channelID   string
	channelType uint8
}

type eventFlags struct {
	clientMsgNo string
	eventKey    string
}

type rawDumpFlags struct {
	tables     []string
	scopes     []string
	decode     bool
	keyPrefix  string
	tailPrefix string
	uidFlags
	channelFlags
	eventFlags
}

type tableSummary struct {
	DataDir     string                `json:"data_dir"`
	BucketCount uint32                `json:"bucket_count,omitempty"`
	Tables      []wkinspect.TableInfo `json:"tables"`
}

func newDbCMD(ctx *WuKongIMContext) *dbCMD {
	return &dbCMD{ctx: ctx}
}

func (d *dbCMD) CMD() *cobra.Command {
	opts := &dbCommandOptions{format: "json", limit: 100}
	cmd := &cobra.Command{
		Use:   "db",
		Short: "inspect wkdb/v3 data on local disk",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	cmd.PersistentFlags().StringVar(&opts.dataDir, "data-dir", "", "wkdb/v3 data directory (defaults to <dataDir>/dbv3 from config)")
	cmd.PersistentFlags().StringVar(&opts.format, "format", "json", "output format: json or table")
	cmd.PersistentFlags().IntVar(&opts.limit, "limit", 100, "max rows to return for list/raw commands; 0 means no limit")
	cmd.PersistentFlags().IntVar(&opts.slotCount, "slot-count", 0, "cluster slot count used to infer slot from uid/channel-id; defaults to cluster.slotCount from config")

	cmd.AddCommand(d.newDBTablesCmd(opts))
	cmd.AddCommand(d.newDBGetCmd(opts))
	cmd.AddCommand(d.newDBListCmd(opts))
	cmd.AddCommand(d.newDBRawCmd(opts))
	return cmd
}

func (d *dbCMD) newDBTablesCmd(opts *dbCommandOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "tables",
		Short: "list inspectable wkdb/v3 tables",
		RunE: func(cmd *cobra.Command, args []string) error {
			dataDir, err := d.resolveDataDir(opts)
			if err != nil {
				return err
			}
			summary := tableSummary{DataDir: dataDir, Tables: wkinspect.Tables()}
			if bucketCount, err := wkinspect.DetectBucketCount(dataDir); err == nil {
				summary.BucketCount = bucketCount
			}
			return renderDBOutput(cmd, opts.format, summary, dbTable{
				headers: []string{"table", "commands", "description"},
				rows: func() [][]string {
					rows := make([][]string, 0, len(summary.Tables))
					for _, info := range summary.Tables {
						rows = append(rows, []string{info.Name, strings.Join(info.Commands, ","), info.Description})
					}
					return rows
				}(),
			})
		},
	}
}

func (d *dbCMD) newDBGetCmd(opts *dbCommandOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "get a single wkdb/v3 record",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(d.newDBGetUserCmd(opts))
	cmd.AddCommand(d.newDBGetChannelCmd(opts))
	cmd.AddCommand(d.newDBGetChannelClusterConfigCmd(opts))
	cmd.AddCommand(d.newDBGetMessageEventStateCmd(opts))
	return cmd
}

func (d *dbCMD) newDBListCmd(opts *dbCommandOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "list wkdb/v3 records",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(d.newDBListUsersCmd(opts))
	cmd.AddCommand(d.newDBListDevicesCmd(opts))
	cmd.AddCommand(d.newDBListConversationsCmd(opts))
	cmd.AddCommand(d.newDBListChannelsCmd(opts))
	cmd.AddCommand(d.newDBListSubscribersCmd(opts))
	cmd.AddCommand(d.newDBListAllowlistsCmd(opts))
	cmd.AddCommand(d.newDBListDenylistsCmd(opts))
	cmd.AddCommand(d.newDBListChannelClusterConfigsCmd(opts))
	return cmd
}

func (d *dbCMD) newDBRawCmd(opts *dbCommandOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "raw",
		Short: "inspect raw wkdb/v3 key/value records",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(d.newDBRawDumpCmd(opts))
	return cmd
}

func (d *dbCMD) newDBGetUserCmd(opts *dbCommandOptions) *cobra.Command {
	var slot slotFlags
	var uid uidFlags
	cmd := &cobra.Command{
		Use:   "user",
		Short: "get a user row by slot and uid",
		RunE: func(cmd *cobra.Command, args []string) error {
			inspector, err := d.openInspector(opts)
			if err != nil {
				return err
			}
			defer inspector.Close()
			slots, err := d.resolveSelectedSlots(context.Background(), inspector, opts, slot, slotRouteHint{key: uid.uid, label: "uid"})
			if err != nil {
				return err
			}
			if len(slots) == 1 {
				user, err := inspector.GetUser(slots[0], uid.uid)
				if err != nil {
					return err
				}
				return renderDBOutput(cmd, opts.format, user, tableForUsers([]wkdb.User{user}))
			}
			users, err := findSlotValues(slots, inspector, func(slotID uint32) (wkdb.User, bool, error) {
				user, err := inspector.GetUser(slotID, uid.uid)
				if errors.Is(err, wkdb.ErrNotFound) {
					return wkdb.User{}, false, nil
				}
				if err != nil {
					return wkdb.User{}, false, err
				}
				return user, true, nil
			})
			if err != nil {
				return err
			}
			return renderDBOutput(cmd, opts.format, users, tableForSlotUsers(users))
		},
	}
	bindSlotFlag(cmd, &slot)
	bindUIDFlag(cmd, &uid, true)
	return cmd
}

func (d *dbCMD) newDBGetChannelCmd(opts *dbCommandOptions) *cobra.Command {
	var slot slotFlags
	var channel channelFlags
	cmd := &cobra.Command{
		Use:   "channel",
		Short: "get a channel row by slot and channel key",
		RunE: func(cmd *cobra.Command, args []string) error {
			inspector, err := d.openInspector(opts)
			if err != nil {
				return err
			}
			defer inspector.Close()
			slots, err := d.resolveSelectedSlots(context.Background(), inspector, opts, slot, slotRouteHint{key: channel.channelID, label: "channel-id"})
			if err != nil {
				return err
			}
			if len(slots) == 1 {
				info, err := inspector.GetChannel(slots[0], channel.channelID, channel.channelType)
				if err != nil {
					return err
				}
				return renderDBOutput(cmd, opts.format, info, tableForChannels([]wkdb.ChannelInfo{info}))
			}
			channels, err := findSlotValues(slots, inspector, func(slotID uint32) (wkdb.ChannelInfo, bool, error) {
				info, err := inspector.GetChannel(slotID, channel.channelID, channel.channelType)
				if errors.Is(err, wkdb.ErrNotFound) {
					return wkdb.ChannelInfo{}, false, nil
				}
				if err != nil {
					return wkdb.ChannelInfo{}, false, err
				}
				return info, true, nil
			})
			if err != nil {
				return err
			}
			return renderDBOutput(cmd, opts.format, channels, tableForSlotChannels(channels))
		},
	}
	bindSlotFlag(cmd, &slot)
	bindChannelFlags(cmd, &channel, true)
	return cmd
}

func (d *dbCMD) newDBGetChannelClusterConfigCmd(opts *dbCommandOptions) *cobra.Command {
	var slot slotFlags
	var channel channelFlags
	cmd := &cobra.Command{
		Use:   "channel-cluster-config",
		Short: "get a channel cluster config row",
		RunE: func(cmd *cobra.Command, args []string) error {
			inspector, err := d.openInspector(opts)
			if err != nil {
				return err
			}
			defer inspector.Close()
			slots, err := d.resolveSelectedSlots(context.Background(), inspector, opts, slot, slotRouteHint{key: channel.channelID, label: "channel-id"})
			if err != nil {
				return err
			}
			if len(slots) == 1 {
				cfg, err := inspector.GetChannelClusterConfig(slots[0], channel.channelID, channel.channelType)
				if err != nil {
					return err
				}
				return renderDBOutput(cmd, opts.format, cfg, tableForChannelClusterConfigs([]wkdb.ChannelClusterConfig{cfg}))
			}
			configs, err := findSlotValues(slots, inspector, func(slotID uint32) (wkdb.ChannelClusterConfig, bool, error) {
				cfg, err := inspector.GetChannelClusterConfig(slotID, channel.channelID, channel.channelType)
				if errors.Is(err, wkdb.ErrNotFound) {
					return wkdb.ChannelClusterConfig{}, false, nil
				}
				if err != nil {
					return wkdb.ChannelClusterConfig{}, false, err
				}
				return cfg, true, nil
			})
			if err != nil {
				return err
			}
			return renderDBOutput(cmd, opts.format, configs, tableForSlotChannelClusterConfigs(configs))
		},
	}
	bindSlotFlag(cmd, &slot)
	bindChannelFlags(cmd, &channel, true)
	return cmd
}

func (d *dbCMD) newDBGetMessageEventStateCmd(opts *dbCommandOptions) *cobra.Command {
	var slot slotFlags
	var channel channelFlags
	var event eventFlags
	cmd := &cobra.Command{
		Use:   "message-event-state",
		Short: "get a message event state row",
		RunE: func(cmd *cobra.Command, args []string) error {
			inspector, err := d.openInspector(opts)
			if err != nil {
				return err
			}
			defer inspector.Close()
			slots, err := d.resolveSelectedSlots(context.Background(), inspector, opts, slot, slotRouteHint{key: channel.channelID, label: "channel-id"})
			if err != nil {
				return err
			}
			if len(slots) == 1 {
				state, err := inspector.GetMessageEventState(slots[0], channel.channelID, channel.channelType, event.clientMsgNo, event.eventKey)
				if err != nil {
					return err
				}
				return renderDBOutput(cmd, opts.format, state, tableForMessageEventStates([]*wkdb.MessageEventState{state}))
			}
			states, err := findSlotValues(slots, inspector, func(slotID uint32) (*wkdb.MessageEventState, bool, error) {
				state, err := inspector.GetMessageEventState(slotID, channel.channelID, channel.channelType, event.clientMsgNo, event.eventKey)
				if err != nil {
					return nil, false, err
				}
				if state == nil {
					return nil, false, nil
				}
				return state, true, nil
			})
			if err != nil {
				return err
			}
			return renderDBOutput(cmd, opts.format, states, tableForSlotMessageEventStates(states))
		},
	}
	bindSlotFlag(cmd, &slot)
	bindChannelFlags(cmd, &channel, true)
	cmd.Flags().StringVar(&event.clientMsgNo, "client-msg-no", "", "client message number")
	cmd.Flags().StringVar(&event.eventKey, "event-key", wkdb.EventKeyDefault, "event key")
	mustMarkFlagRequired(cmd, "client-msg-no")
	return cmd
}

func (d *dbCMD) newDBListUsersCmd(opts *dbCommandOptions) *cobra.Command {
	var slot slotFlags
	cmd := &cobra.Command{
		Use:   "users",
		Short: "list users in a slot",
		RunE: func(cmd *cobra.Command, args []string) error {
			inspector, err := d.openInspector(opts)
			if err != nil {
				return err
			}
			defer inspector.Close()
			slots, err := d.resolveSelectedSlots(context.Background(), inspector, opts, slot, slotRouteHint{})
			if err != nil {
				return err
			}
			if len(slots) == 1 {
				users, err := inspector.ListUsers(slots[0], opts.limit)
				if err != nil {
					return err
				}
				return renderDBOutput(cmd, opts.format, users, tableForUsers(users))
			}
			users, err := collectSlotValues(slots, inspector, opts.limit, inspector.ListUsers)
			if err != nil {
				return err
			}
			return renderDBOutput(cmd, opts.format, users, tableForSlotUsers(users))
		},
	}
	bindSlotFlag(cmd, &slot)
	return cmd
}

func (d *dbCMD) newDBListDevicesCmd(opts *dbCommandOptions) *cobra.Command {
	var slot slotFlags
	var uid uidFlags
	cmd := &cobra.Command{
		Use:   "devices",
		Short: "list devices in a slot or under a uid",
		RunE: func(cmd *cobra.Command, args []string) error {
			inspector, err := d.openInspector(opts)
			if err != nil {
				return err
			}
			defer inspector.Close()
			slots, err := d.resolveSelectedSlots(context.Background(), inspector, opts, slot, slotRouteHint{key: uid.uid, label: "uid"})
			if err != nil {
				return err
			}
			if len(slots) == 1 {
				devices, err := inspector.ListDevices(slots[0], uid.uid, opts.limit)
				if err != nil {
					return err
				}
				return renderDBOutput(cmd, opts.format, devices, tableForDevices(devices))
			}
			devices, err := collectSlotValues(slots, inspector, opts.limit, func(slotID uint32, currentLimit int) ([]wkdb.Device, error) {
				return inspector.ListDevices(slotID, uid.uid, currentLimit)
			})
			if err != nil {
				return err
			}
			return renderDBOutput(cmd, opts.format, devices, tableForSlotDevices(devices))
		},
	}
	bindSlotFlag(cmd, &slot)
	bindUIDFlag(cmd, &uid, false)
	return cmd
}

func (d *dbCMD) newDBListConversationsCmd(opts *dbCommandOptions) *cobra.Command {
	var slot slotFlags
	var uid uidFlags
	cmd := &cobra.Command{
		Use:   "conversations",
		Short: "list conversations in a slot or under a uid",
		RunE: func(cmd *cobra.Command, args []string) error {
			inspector, err := d.openInspector(opts)
			if err != nil {
				return err
			}
			defer inspector.Close()
			slots, err := d.resolveSelectedSlots(context.Background(), inspector, opts, slot, slotRouteHint{key: uid.uid, label: "uid"})
			if err != nil {
				return err
			}
			if len(slots) == 1 {
				conversations, err := inspector.ListConversations(slots[0], uid.uid, opts.limit)
				if err != nil {
					return err
				}
				return renderDBOutput(cmd, opts.format, conversations, tableForConversations(conversations))
			}
			conversations, err := collectSlotValues(slots, inspector, opts.limit, func(slotID uint32, currentLimit int) ([]wkdb.Conversation, error) {
				return inspector.ListConversations(slotID, uid.uid, currentLimit)
			})
			if err != nil {
				return err
			}
			return renderDBOutput(cmd, opts.format, conversations, tableForSlotConversations(conversations))
		},
	}
	bindSlotFlag(cmd, &slot)
	bindUIDFlag(cmd, &uid, false)
	return cmd
}

func (d *dbCMD) newDBListChannelsCmd(opts *dbCommandOptions) *cobra.Command {
	var slot slotFlags
	cmd := &cobra.Command{
		Use:   "channels",
		Short: "list channels in a slot",
		RunE: func(cmd *cobra.Command, args []string) error {
			inspector, err := d.openInspector(opts)
			if err != nil {
				return err
			}
			defer inspector.Close()
			slots, err := d.resolveSelectedSlots(context.Background(), inspector, opts, slot, slotRouteHint{})
			if err != nil {
				return err
			}
			if len(slots) == 1 {
				channels, err := inspector.ListChannels(slots[0], opts.limit)
				if err != nil {
					return err
				}
				return renderDBOutput(cmd, opts.format, channels, tableForChannels(channels))
			}
			channels, err := collectSlotValues(slots, inspector, opts.limit, inspector.ListChannels)
			if err != nil {
				return err
			}
			return renderDBOutput(cmd, opts.format, channels, tableForSlotChannels(channels))
		},
	}
	bindSlotFlag(cmd, &slot)
	return cmd
}

func (d *dbCMD) newDBListSubscribersCmd(opts *dbCommandOptions) *cobra.Command {
	var slot slotFlags
	var channel channelFlags
	cmd := &cobra.Command{
		Use:   "subscribers",
		Short: "list subscriber members for a channel",
		RunE: func(cmd *cobra.Command, args []string) error {
			inspector, err := d.openInspector(opts)
			if err != nil {
				return err
			}
			defer inspector.Close()
			slots, err := d.resolveSelectedSlots(context.Background(), inspector, opts, slot, slotRouteHint{key: channel.channelID, label: "channel-id"})
			if err != nil {
				return err
			}
			if len(slots) == 1 {
				members, err := inspector.ListSubscribers(slots[0], channel.channelID, channel.channelType)
				if err != nil {
					return err
				}
				return renderDBOutput(cmd, opts.format, members, tableForMembers(members))
			}
			members, err := collectSlotValues(slots, inspector, opts.limit, func(slotID uint32, currentLimit int) ([]wkdb.Member, error) {
				items, err := inspector.ListSubscribers(slotID, channel.channelID, channel.channelType)
				if err != nil {
					return nil, err
				}
				return limitMembers(items, currentLimit), nil
			})
			if err != nil {
				return err
			}
			return renderDBOutput(cmd, opts.format, members, tableForSlotMembers(members))
		},
	}
	bindSlotFlag(cmd, &slot)
	bindChannelFlags(cmd, &channel, true)
	return cmd
}

func (d *dbCMD) newDBListAllowlistsCmd(opts *dbCommandOptions) *cobra.Command {
	var slot slotFlags
	var channel channelFlags
	cmd := &cobra.Command{
		Use:   "allowlists",
		Short: "list allowlist members for a channel",
		RunE: func(cmd *cobra.Command, args []string) error {
			inspector, err := d.openInspector(opts)
			if err != nil {
				return err
			}
			defer inspector.Close()
			slots, err := d.resolveSelectedSlots(context.Background(), inspector, opts, slot, slotRouteHint{key: channel.channelID, label: "channel-id"})
			if err != nil {
				return err
			}
			if len(slots) == 1 {
				members, err := inspector.ListAllowlists(slots[0], channel.channelID, channel.channelType)
				if err != nil {
					return err
				}
				return renderDBOutput(cmd, opts.format, members, tableForMembers(members))
			}
			members, err := collectSlotValues(slots, inspector, opts.limit, func(slotID uint32, currentLimit int) ([]wkdb.Member, error) {
				items, err := inspector.ListAllowlists(slotID, channel.channelID, channel.channelType)
				if err != nil {
					return nil, err
				}
				return limitMembers(items, currentLimit), nil
			})
			if err != nil {
				return err
			}
			return renderDBOutput(cmd, opts.format, members, tableForSlotMembers(members))
		},
	}
	bindSlotFlag(cmd, &slot)
	bindChannelFlags(cmd, &channel, true)
	return cmd
}

func (d *dbCMD) newDBListDenylistsCmd(opts *dbCommandOptions) *cobra.Command {
	var slot slotFlags
	var channel channelFlags
	cmd := &cobra.Command{
		Use:   "denylists",
		Short: "list denylist members for a channel",
		RunE: func(cmd *cobra.Command, args []string) error {
			inspector, err := d.openInspector(opts)
			if err != nil {
				return err
			}
			defer inspector.Close()
			slots, err := d.resolveSelectedSlots(context.Background(), inspector, opts, slot, slotRouteHint{key: channel.channelID, label: "channel-id"})
			if err != nil {
				return err
			}
			if len(slots) == 1 {
				members, err := inspector.ListDenylists(slots[0], channel.channelID, channel.channelType)
				if err != nil {
					return err
				}
				return renderDBOutput(cmd, opts.format, members, tableForMembers(members))
			}
			members, err := collectSlotValues(slots, inspector, opts.limit, func(slotID uint32, currentLimit int) ([]wkdb.Member, error) {
				items, err := inspector.ListDenylists(slotID, channel.channelID, channel.channelType)
				if err != nil {
					return nil, err
				}
				return limitMembers(items, currentLimit), nil
			})
			if err != nil {
				return err
			}
			return renderDBOutput(cmd, opts.format, members, tableForSlotMembers(members))
		},
	}
	bindSlotFlag(cmd, &slot)
	bindChannelFlags(cmd, &channel, true)
	return cmd
}

func (d *dbCMD) newDBListChannelClusterConfigsCmd(opts *dbCommandOptions) *cobra.Command {
	var slot slotFlags
	cmd := &cobra.Command{
		Use:   "channel-cluster-configs",
		Short: "list channel cluster configs in a slot",
		RunE: func(cmd *cobra.Command, args []string) error {
			inspector, err := d.openInspector(opts)
			if err != nil {
				return err
			}
			defer inspector.Close()
			slots, err := d.resolveSelectedSlots(context.Background(), inspector, opts, slot, slotRouteHint{})
			if err != nil {
				return err
			}
			if len(slots) == 1 {
				configs, err := inspector.ListChannelClusterConfigs(slots[0], opts.limit)
				if err != nil {
					return err
				}
				return renderDBOutput(cmd, opts.format, configs, tableForChannelClusterConfigs(configs))
			}
			configs, err := collectSlotValues(slots, inspector, opts.limit, inspector.ListChannelClusterConfigs)
			if err != nil {
				return err
			}
			return renderDBOutput(cmd, opts.format, configs, tableForSlotChannelClusterConfigs(configs))
		},
	}
	bindSlotFlag(cmd, &slot)
	return cmd
}

func (d *dbCMD) newDBRawDumpCmd(opts *dbCommandOptions) *cobra.Command {
	var slot slotFlags
	flags := rawDumpFlags{decode: true}
	cmd := &cobra.Command{
		Use:   "dump",
		Short: "dump raw key/value records for a slot",
		RunE: func(cmd *cobra.Command, args []string) error {
			inspector, err := d.openInspector(opts)
			if err != nil {
				return err
			}
			defer inspector.Close()
			slots, err := d.resolveSelectedSlots(context.Background(), inspector, opts, slot, rawDumpRouteHint(flags))
			if err != nil {
				return err
			}
			rawOpts, err := parseRawDumpOptions(cmd, opts.limit, flags)
			if err != nil {
				return err
			}
			records := make([]wkinspect.RawKVRecord, 0)
			remaining := rawOpts.Limit
			for _, slotID := range slots {
				slotOpts := rawOpts
				slotOpts.Limit = 0
				if rawOpts.Limit > 0 {
					if remaining <= 0 {
						break
					}
					slotOpts.Limit = remaining
				}
				items, err := inspector.RawDumpSlotWithOptions(context.Background(), slotID, slotOpts)
				if err != nil {
					return err
				}
				records = append(records, items...)
				if rawOpts.Limit > 0 {
					remaining -= len(items)
				}
			}
			return renderDBOutput(cmd, opts.format, records, tableForRawRecords(records))
		},
	}
	bindSlotFlag(cmd, &slot)
	cmd.Flags().StringSliceVar(&flags.tables, "table", nil, "filter tables (comma-separated), for example: user,conversation,channel")
	cmd.Flags().StringSliceVar(&flags.scopes, "scope", nil, "filter scopes (comma-separated), for example: primary,second_index,aux")
	cmd.Flags().StringVar(&flags.keyPrefix, "key-prefix", "", "filter by full key hex prefix, for example: 01010000000712")
	cmd.Flags().StringVar(&flags.tailPrefix, "tail-prefix", "", "filter by decoded tail hex prefix, for example: 00000006757365722d31")
	cmd.Flags().BoolVar(&flags.decode, "decode", true, "decode known raw values")
	bindUIDFlag(cmd, &flags.uidFlags, false)
	bindChannelFlags(cmd, &flags.channelFlags, false)
	bindEventFlags(cmd, &flags.eventFlags, false)
	return cmd
}

func (d *dbCMD) openInspector(opts *dbCommandOptions) (*wkinspect.Inspector, error) {
	dataDir, err := d.resolveDataDir(opts)
	if err != nil {
		return nil, err
	}
	return wkinspect.Open(wkinspect.Options{DataDir: dataDir})
}

func (d *dbCMD) resolveDataDir(opts *dbCommandOptions) (string, error) {
	if opts != nil && strings.TrimSpace(opts.dataDir) != "" {
		return opts.dataDir, nil
	}
	if serverOpts != nil && strings.TrimSpace(serverOpts.DataDir) != "" {
		return filepath.Join(serverOpts.DataDir, "dbv3"), nil
	}
	return "", fmt.Errorf("db inspect requires --data-dir or a configured dataDir")
}

func renderDBOutput(cmd *cobra.Command, format string, payload any, table dbTable) error {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", "json":
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(payload)
	case "table":
		return renderDBTable(cmd, table)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

func parseRawDumpOptions(cmd *cobra.Command, limit int, flags rawDumpFlags) (wkinspect.RawDumpOptions, error) {
	tables, err := parseRawDumpTables(flags.tables)
	if err != nil {
		return wkinspect.RawDumpOptions{}, err
	}
	scopes, err := parseRawDumpScopes(flags.scopes)
	if err != nil {
		return wkinspect.RawDumpOptions{}, err
	}
	keyPrefix, err := parseHexPrefix(flags.keyPrefix)
	if err != nil {
		return wkinspect.RawDumpOptions{}, fmt.Errorf("parse --key-prefix: %w", err)
	}
	tailPrefix, err := parseHexPrefix(flags.tailPrefix)
	if err != nil {
		return wkinspect.RawDumpOptions{}, fmt.Errorf("parse --tail-prefix: %w", err)
	}
	var channelType *uint8
	if cmd != nil && cmd.Flags().Changed("channel-type") {
		v := flags.channelType
		channelType = &v
	}
	return wkinspect.RawDumpOptions{
		Limit:  limit,
		Tables: tables,
		Scopes: scopes,
		KeyFilter: wkinspect.RawKeyFilter{
			UID:         strings.TrimSpace(flags.uid),
			ChannelID:   strings.TrimSpace(flags.channelID),
			ChannelType: channelType,
			ClientMsgNo: strings.TrimSpace(flags.clientMsgNo),
			EventKey:    strings.TrimSpace(flags.eventKey),
			KeyPrefix:   keyPrefix,
			TailPrefix:  tailPrefix,
		},
		DecodeValues: flags.decode,
	}, nil
}

func parseHexPrefix(raw string) ([]byte, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	raw = strings.TrimPrefix(strings.ToLower(raw), "0x")
	raw = strings.ReplaceAll(raw, " ", "")
	raw = strings.ReplaceAll(raw, "_", "")
	if len(raw)%2 != 0 {
		return nil, fmt.Errorf("hex prefix must have even length")
	}
	decoded, err := hex.DecodeString(raw)
	if err != nil {
		return nil, err
	}
	return decoded, nil
}

func rawDumpRouteHint(flags rawDumpFlags) slotRouteHint {
	if strings.TrimSpace(flags.uid) != "" && strings.TrimSpace(flags.channelID) == "" {
		return slotRouteHint{key: flags.uid, label: "uid"}
	}
	if strings.TrimSpace(flags.channelID) != "" && strings.TrimSpace(flags.uid) == "" {
		return slotRouteHint{key: flags.channelID, label: "channel-id"}
	}
	return slotRouteHint{}
}

func parseRawDumpTables(values []string) ([]v3key.TableID, error) {
	return parseRawDumpFilter(values, func(value string) (v3key.TableID, error) {
		return v3key.ParseTableID(value)
	})
}

func parseRawDumpScopes(values []string) ([]v3key.ScopeType, error) {
	return parseRawDumpFilter(values, func(value string) (v3key.ScopeType, error) {
		return v3key.ParseScopeType(value)
	})
}

func parseRawDumpFilter[T comparable](values []string, parse func(string) (T, error)) ([]T, error) {
	if len(values) == 0 {
		return nil, nil
	}
	seen := make(map[T]struct{})
	out := make([]T, 0, len(values))
	for _, value := range values {
		for _, token := range strings.Split(value, ",") {
			token = strings.TrimSpace(token)
			if token == "" {
				continue
			}
			if strings.EqualFold(token, "all") {
				return nil, nil
			}
			parsed, err := parse(token)
			if err != nil {
				return nil, err
			}
			if _, ok := seen[parsed]; ok {
				continue
			}
			seen[parsed] = struct{}{}
			out = append(out, parsed)
		}
	}
	return out, nil
}

func renderDBTable(cmd *cobra.Command, table dbTable) error {
	if len(table.headers) == 0 {
		return fmt.Errorf("table output requires headers")
	}
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 8, 2, ' ', 0)
	if _, err := fmt.Fprintln(w, strings.Join(table.headers, "\t")); err != nil {
		return err
	}
	if len(table.rows) == 0 {
		if _, err := fmt.Fprintln(w, "(empty)"); err != nil {
			return err
		}
		return w.Flush()
	}
	for _, row := range table.rows {
		if _, err := fmt.Fprintln(w, strings.Join(row, "\t")); err != nil {
			return err
		}
	}
	return w.Flush()
}

func bindSlotFlag(cmd *cobra.Command, flags *slotFlags) {
	cmd.Flags().StringVar(&flags.selector, "slot", "", "slot selector: <id>, <start-end>, comma list, or all (optional when uid/channel-id can infer routing)")
}

func bindUIDFlag(cmd *cobra.Command, flags *uidFlags, required bool) {
	cmd.Flags().StringVar(&flags.uid, "uid", "", "user id")
	if required {
		mustMarkFlagRequired(cmd, "uid")
	}
}

func bindChannelFlags(cmd *cobra.Command, flags *channelFlags, required bool) {
	cmd.Flags().StringVar(&flags.channelID, "channel-id", "", "channel id")
	cmd.Flags().Uint8Var(&flags.channelType, "channel-type", 0, "channel type")
	if required {
		mustMarkFlagRequired(cmd, "channel-id")
		mustMarkFlagRequired(cmd, "channel-type")
	}
}

func bindEventFlags(cmd *cobra.Command, flags *eventFlags, required bool) {
	cmd.Flags().StringVar(&flags.clientMsgNo, "client-msg-no", "", "client message no")
	cmd.Flags().StringVar(&flags.eventKey, "event-key", "", "message event key")
	if required {
		mustMarkFlagRequired(cmd, "client-msg-no")
		mustMarkFlagRequired(cmd, "event-key")
	}
}

func mustMarkFlagRequired(cmd *cobra.Command, name string) {
	if err := cmd.MarkFlagRequired(name); err != nil {
		panic(err)
	}
}

func tableForUsers(users []wkdb.User) dbTable {
	rows := make([][]string, 0, len(users))
	for _, user := range users {
		rows = append(rows, userRow(user))
	}
	return dbTable{headers: []string{"id", "uid", "plugin_no", "created_at", "updated_at"}, rows: rows}
}

func tableForDevices(devices []wkdb.Device) dbTable {
	rows := make([][]string, 0, len(devices))
	for _, device := range devices {
		rows = append(rows, deviceRow(device))
	}
	return dbTable{headers: []string{"id", "uid", "device_flag", "device_level", "token", "created_at", "updated_at"}, rows: rows}
}

func tableForConversations(conversations []wkdb.Conversation) dbTable {
	rows := make([][]string, 0, len(conversations))
	for _, conversation := range conversations {
		rows = append(rows, conversationRow(conversation))
	}
	return dbTable{headers: []string{"id", "uid", "channel_id", "channel_type", "type", "read_to_msg_seq", "deleted_at_msg_seq", "created_at", "updated_at"}, rows: rows}
}

func tableForChannels(channels []wkdb.ChannelInfo) dbTable {
	rows := make([][]string, 0, len(channels))
	for _, channel := range channels {
		rows = append(rows, channelRow(channel))
	}
	return dbTable{headers: []string{"id", "channel_id", "channel_type", "ban", "large", "disband", "subscriber_count", "allowlist_count", "denylist_count", "created_at", "updated_at"}, rows: rows}
}

func tableForMembers(members []wkdb.Member) dbTable {
	rows := make([][]string, 0, len(members))
	for _, member := range members {
		rows = append(rows, memberRow(member))
	}
	return dbTable{headers: []string{"id", "uid", "created_at", "updated_at"}, rows: rows}
}

func tableForChannelClusterConfigs(configs []wkdb.ChannelClusterConfig) dbTable {
	rows := make([][]string, 0, len(configs))
	for _, cfg := range configs {
		rows = append(rows, channelClusterConfigRow(cfg))
	}
	return dbTable{headers: []string{"id", "channel_id", "channel_type", "leader_id", "conf_version", "replica_max_count", "replicas", "learners", "created_at", "updated_at"}, rows: rows}
}

func tableForMessageEventStates(states []*wkdb.MessageEventState) dbTable {
	rows := make([][]string, 0, len(states))
	for _, state := range states {
		if state == nil {
			continue
		}
		rows = append(rows, messageEventStateRow(state))
	}
	return dbTable{headers: []string{"channel_id", "channel_type", "client_msg_no", "event_key", "last_msg_event_seq", "snapshot_payload_size"}, rows: rows}
}

func tableForRawRecords(records []wkinspect.RawKVRecord) dbTable {
	rows := make([][]string, 0, len(records))
	for _, record := range records {
		rows = append(rows, rawRecordRow(record))
	}
	return dbTable{headers: []string{"index", "slot", "bucket", "scope", "table", "kind", "value_size", "key_hex", "tail_hex", "key_decoded", "decoded"}, rows: rows}
}

func tableForSlotUsers(users []slotScopedValue[wkdb.User]) dbTable {
	rows := make([][]string, 0, len(users))
	for _, user := range users {
		rows = append(rows, prependSlotContext(user.SlotID, user.BucketID, userRow(user.Data)))
	}
	return dbTable{headers: append(slotContextHeaders(), "id", "uid", "plugin_no", "created_at", "updated_at"), rows: rows}
}

func tableForSlotDevices(devices []slotScopedValue[wkdb.Device]) dbTable {
	rows := make([][]string, 0, len(devices))
	for _, device := range devices {
		rows = append(rows, prependSlotContext(device.SlotID, device.BucketID, deviceRow(device.Data)))
	}
	return dbTable{headers: append(slotContextHeaders(), "id", "uid", "device_flag", "device_level", "token", "created_at", "updated_at"), rows: rows}
}

func tableForSlotConversations(conversations []slotScopedValue[wkdb.Conversation]) dbTable {
	rows := make([][]string, 0, len(conversations))
	for _, conversation := range conversations {
		rows = append(rows, prependSlotContext(conversation.SlotID, conversation.BucketID, conversationRow(conversation.Data)))
	}
	return dbTable{headers: append(slotContextHeaders(), "id", "uid", "channel_id", "channel_type", "type", "read_to_msg_seq", "deleted_at_msg_seq", "created_at", "updated_at"), rows: rows}
}

func tableForSlotChannels(channels []slotScopedValue[wkdb.ChannelInfo]) dbTable {
	rows := make([][]string, 0, len(channels))
	for _, channel := range channels {
		rows = append(rows, prependSlotContext(channel.SlotID, channel.BucketID, channelRow(channel.Data)))
	}
	return dbTable{headers: append(slotContextHeaders(), "id", "channel_id", "channel_type", "ban", "large", "disband", "subscriber_count", "allowlist_count", "denylist_count", "created_at", "updated_at"), rows: rows}
}

func tableForSlotMembers(members []slotScopedValue[wkdb.Member]) dbTable {
	rows := make([][]string, 0, len(members))
	for _, member := range members {
		rows = append(rows, prependSlotContext(member.SlotID, member.BucketID, memberRow(member.Data)))
	}
	return dbTable{headers: append(slotContextHeaders(), "id", "uid", "created_at", "updated_at"), rows: rows}
}

func tableForSlotChannelClusterConfigs(configs []slotScopedValue[wkdb.ChannelClusterConfig]) dbTable {
	rows := make([][]string, 0, len(configs))
	for _, cfg := range configs {
		rows = append(rows, prependSlotContext(cfg.SlotID, cfg.BucketID, channelClusterConfigRow(cfg.Data)))
	}
	return dbTable{headers: append(slotContextHeaders(), "id", "channel_id", "channel_type", "leader_id", "conf_version", "replica_max_count", "replicas", "learners", "created_at", "updated_at"), rows: rows}
}

func tableForSlotMessageEventStates(states []slotScopedValue[*wkdb.MessageEventState]) dbTable {
	rows := make([][]string, 0, len(states))
	for _, state := range states {
		if state.Data == nil {
			continue
		}
		rows = append(rows, prependSlotContext(state.SlotID, state.BucketID, messageEventStateRow(state.Data)))
	}
	return dbTable{headers: append(slotContextHeaders(), "channel_id", "channel_type", "client_msg_no", "event_key", "last_msg_event_seq", "snapshot_payload_size"), rows: rows}
}

func slotContextHeaders() []string {
	return []string{"slot", "bucket"}
}

func prependSlotContext(slotID, bucketID uint32, row []string) []string {
	return append([]string{strconv.FormatUint(uint64(slotID), 10), strconv.FormatUint(uint64(bucketID), 10)}, row...)
}

func userRow(user wkdb.User) []string {
	return []string{
		strconv.FormatUint(user.Id, 10),
		user.Uid,
		user.PluginNo,
		formatTime(user.CreatedAt),
		formatTime(user.UpdatedAt),
	}
}

func deviceRow(device wkdb.Device) []string {
	return []string{
		strconv.FormatUint(device.Id, 10),
		device.Uid,
		strconv.FormatUint(device.DeviceFlag, 10),
		strconv.Itoa(int(device.DeviceLevel)),
		device.Token,
		formatTime(device.CreatedAt),
		formatTime(device.UpdatedAt),
	}
}

func conversationRow(conversation wkdb.Conversation) []string {
	return []string{
		strconv.FormatUint(conversation.Id, 10),
		conversation.Uid,
		conversation.ChannelId,
		strconv.Itoa(int(conversation.ChannelType)),
		strconv.Itoa(int(conversation.Type)),
		strconv.FormatUint(conversation.ReadToMsgSeq, 10),
		strconv.FormatUint(conversation.DeletedAtMsgSeq, 10),
		formatTime(conversation.CreatedAt),
		formatTime(conversation.UpdatedAt),
	}
}

func channelRow(channel wkdb.ChannelInfo) []string {
	return []string{
		strconv.FormatUint(channel.Id, 10),
		channel.ChannelId,
		strconv.Itoa(int(channel.ChannelType)),
		strconv.FormatBool(channel.Ban),
		strconv.FormatBool(channel.Large),
		strconv.FormatBool(channel.Disband),
		strconv.Itoa(channel.SubscriberCount),
		strconv.Itoa(channel.AllowlistCount),
		strconv.Itoa(channel.DenylistCount),
		formatTime(channel.CreatedAt),
		formatTime(channel.UpdatedAt),
	}
}

func memberRow(member wkdb.Member) []string {
	return []string{
		strconv.FormatUint(member.Id, 10),
		member.Uid,
		formatTime(member.CreatedAt),
		formatTime(member.UpdatedAt),
	}
}

func channelClusterConfigRow(cfg wkdb.ChannelClusterConfig) []string {
	return []string{
		strconv.FormatUint(cfg.Id, 10),
		cfg.ChannelId,
		strconv.Itoa(int(cfg.ChannelType)),
		strconv.FormatUint(cfg.LeaderId, 10),
		strconv.FormatUint(cfg.ConfVersion, 10),
		strconv.Itoa(int(cfg.ReplicaMaxCount)),
		joinUint64s(cfg.Replicas),
		joinUint64s(cfg.Learners),
		formatTime(cfg.CreatedAt),
		formatTime(cfg.UpdatedAt),
	}
}

func messageEventStateRow(state *wkdb.MessageEventState) []string {
	if state == nil {
		return nil
	}
	return []string{
		state.ChannelId,
		strconv.Itoa(int(state.ChannelType)),
		state.ClientMsgNo,
		state.EventKey,
		strconv.FormatUint(state.LastMsgEventSeq, 10),
		strconv.Itoa(len(state.SnapshotPayload)),
	}
}

func rawRecordRow(record wkinspect.RawKVRecord) []string {
	return []string{
		strconv.Itoa(record.Index),
		strconv.FormatUint(uint64(record.SlotID), 10),
		strconv.FormatUint(uint64(record.BucketID), 10),
		record.Scope,
		record.Table,
		record.Kind,
		strconv.Itoa(record.ValueSize),
		record.KeyHex,
		record.TailHex,
		formatRawDecoded(record.KeyDecoded),
		formatRawDecoded(record.Decoded),
	}
}

func formatRawDecoded(value any) string {
	if value == nil {
		return ""
	}
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprintf("<encode error: %v>", err)
	}
	return string(data)
}

func limitMembers(members []wkdb.Member, limit int) []wkdb.Member {
	if limit <= 0 || len(members) <= limit {
		return members
	}
	return members[:limit]
}

func formatTime(ts *time.Time) string {
	if ts == nil {
		return ""
	}
	return ts.UTC().Format(time.RFC3339Nano)
}

func joinUint64s(values []uint64) string {
	if len(values) == 0 {
		return ""
	}
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, strconv.FormatUint(value, 10))
	}
	return strings.Join(parts, ",")
}
