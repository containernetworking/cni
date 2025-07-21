// Copyright 2015 CNI authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"context"

	"github.com/spf13/cobra"
)

// gcCmd represents the gc command
var gcCmd = &cobra.Command{
	Use:   "gc <network-name> <netns>",
	Short: "Garbage collect network interfaces",
	Long: `Garbage collect network interfaces.
This command will clean up unused network interfaces in the specified network namespace.`,
	Args: cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		netconf, _, err := setupRuntimeConfig(cmd, args)
		if err != nil {
			return err
		}

		cninet := getCNIConfig()
		// Currently just invoke GC without args, hence all network interface should be GC'ed!
		return cninet.GCNetworkList(context.TODO(), netconf, nil)
	},
}

func init() {
	rootCmd.AddCommand(gcCmd)
}
