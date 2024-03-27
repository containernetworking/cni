/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"context"
	"crypto/sha512"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/containernetworking/cni/libcni"
	"github.com/containernetworking/cni/pkg/types"
	"github.com/spf13/cobra"
)

func main() {
	initFlags()

	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(delCmd)
	rootCmd.AddCommand(checkCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(gcCmd)

	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

const (
	EnvCNIPath        = "CNI_PATH"
	EnvNetDir         = "NETCONFPATH"
	EnvCapabilityArgs = "CAP_ARGS"
	EnvCNIArgs        = "CNI_ARGS"
	EnvCNIIfname      = "CNI_IFNAME"
)

var (
	cniBinDir  string
	cniConfDir string

	ifName         string
	capabilityArgs string
	cniArgs        string

	cniBinDirs           []string
	capabilityArgsParsed map[string]interface{}
	cniArgsParsed        [][2]string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:          "cnitool",
	Short:        "execute CNI operations manually",
	SilenceUsage: true,
}

var addCmd = &cobra.Command{
	Use:   "add <net> <netns>",
	Short: "attach a container to a CNI network",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return doAttach(cmd.Context(), "add", args)
	},
}

var delCmd = &cobra.Command{
	Use:   "del <net> <netns>",
	Short: "delete a container's CNI attachment",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return doAttach(cmd.Context(), "del", args)
	},
}

var checkCmd = &cobra.Command{
	Use:   "check <net> <netns>",
	Short: "check a container's CNI attachment",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return doAttach(cmd.Context(), "check", args)
	},
}

var statusCmd = &cobra.Command{
	Use:   "status <net>",
	Short: "check a CNI network's status",
	Args:  cobra.ExactArgs(1),
	RunE:  doStatus,
}

var gcCmd = &cobra.Command{
	Use:   "gc <net> [<netns>[:<ifname>]]...",
	Short: "garbage collect any stale resources",
	Long:  "GR removes any stale resources, keeping any belonging to the set of network namespaces provided",
	Args:  cobra.MinimumNArgs(1),
	RunE:  doGC,
}

// some args are by-environment; pull them in
func loadArgs() error {
	if cniBinDir == "" {
		cniBinDir = os.Getenv(EnvCNIPath)
	}
	if cniBinDir == "" {
		return fmt.Errorf("--cni-bin-dir is required")
	}
	cniBinDirs = filepath.SplitList(cniBinDir)

	if cniConfDir == "" {
		cniConfDir = os.Getenv(EnvNetDir)
	}
	if cniConfDir == "" {
		return fmt.Errorf("--cni-conf-dir is required")
	}

	if capabilityArgs == "" {
		capabilityArgs = os.Getenv(EnvCapabilityArgs)
	}
	if cniArgs == "" {
		cniArgs = os.Getenv(EnvCNIArgs)
	}
	if i := os.Getenv(EnvCNIIfname); i != "" {
		ifName = i
	}

	if len(capabilityArgs) > 0 {
		if err := json.Unmarshal([]byte(capabilityArgs), &capabilityArgs); err != nil {
			return fmt.Errorf("failed to parse capability args: %w", err)
		}
	}

	if len(cniArgs) > 0 {
		for _, pair := range strings.Split(cniArgs, ";") {
			kv := strings.Split(pair, "=")
			if len(kv) != 2 || kv[0] == "" || kv[1] == "" {
				return fmt.Errorf("invalid cni-args pair %q", pair)
			}
			cniArgsParsed = append(cniArgsParsed, [2]string{kv[0], kv[1]})
		}
	}

	return nil
}

// doAttach executes either add, del, or check
func doAttach(ctx context.Context, op string, args []string) error {
	if err := loadArgs(); err != nil {
		return err
	}

	if len(args) != 2 {
		return fmt.Errorf("2 arguments required")
	}

	name := args[0]
	netns := args[1]

	cninet := libcni.NewCNIConfig(cniBinDirs, nil)

	rt := &libcni.RuntimeConf{
		ContainerID:    containerID(netns),
		NetNS:          netns,
		IfName:         ifName,
		Args:           cniArgsParsed,
		CapabilityArgs: capabilityArgsParsed,
	}

	netconf, err := libcni.LoadConfList(cniConfDir, name)
	if err != nil {
		return err
	}

	switch op {
	case "add":
		result, err := cninet.AddNetworkList(ctx, netconf, rt)
		if result != nil {
			_ = result.Print()
		}
		return err
	case "del":
		return cninet.DelNetworkList(ctx, netconf, rt)
	case "check":
		return cninet.CheckNetworkList(ctx, netconf, rt)
	}

	return nil
}

func doStatus(cmd *cobra.Command, args []string) error {
	if err := loadArgs(); err != nil {
		return err
	}

	if len(args) != 1 {
		return fmt.Errorf("1 argument required")
	}
	name := args[0]

	cninet := libcni.NewCNIConfig(cniBinDirs, nil)
	netconf, err := libcni.LoadConfList(cniConfDir, name)
	if err != nil {
		return err
	}

	err = cninet.GetStatusNetworkList(cmd.Context(), netconf)
	if err != nil {
		return fmt.Errorf("network %s is not ready: %w", name, err)
	}
	cmd.Printf("Network %s is ready for ADD requests\n", name)
	return nil
}

func doGC(cmd *cobra.Command, args []string) error {
	if err := loadArgs(); err != nil {
		return err
	}

	if len(args) < 1 {
		return fmt.Errorf("1 argument required")
	}

	validAttachments := []types.GCAttachment{}
	for _, netns := range args[1:] {
		pair := strings.Split(netns, ":")
		ifname := "eth0"
		if len(pair) > 1 {
			ifname = pair[1]
		}

		validAttachments = append(validAttachments, types.GCAttachment{
			IfName:      ifname,
			ContainerID: containerID(netns),
		})
	}

	name := args[0]
	netconf, err := libcni.LoadConfList(cniConfDir, name)
	if err != nil {
		return err
	}

	cninet := libcni.NewCNIConfig(cniBinDirs, nil)
	return cninet.GCNetworkList(cmd.Context(), netconf, &libcni.GCArgs{ValidAttachments: validAttachments})
}

func containerID(netns string) string {
	s := sha512.Sum512([]byte(netns))
	return fmt.Sprintf("cnitool-%x", s[:10])
}

func initFlags() {
	rootCmd.PersistentFlags().StringVar(&cniBinDir, "cni-bin-dir", "", "The folder(s) in which to look for CNI plugins, colon-separated.")
	rootCmd.PersistentFlags().StringVar(&cniConfDir, "cni-conf-dir", "/etc/cni/net.d", "The folder in which to look for CNI network configurations.")

	// common args between ADD, DEL, and CHECK
	for _, cmd := range []*cobra.Command{addCmd, delCmd, checkCmd} {
		cmd.Flags().StringVar(&ifName, "ifname", "eth0", "The value to pass to CNI_IFNAME")
		cmd.Flags().StringVar(&capabilityArgs, "capability-args", "", "Capability args, json-formatted, to pass to the plugins.")
		cmd.Flags().StringVar(&cniArgs, "cni-args", "", "Plugin arguments, in <key>=<v>;... format")
	}
}
