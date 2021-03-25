/*
Copyright © 2021 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"github.com/hfeng101/Sunwukong/cmd/initial"
	"github.com/hfeng101/Sunwukong/util/consts"
	"github.com/spf13/cobra"
)

// ZaohuaCmd represents the Zaohua command
var ZaohuaCmd = &cobra.Command{
	Use:   "zaohua",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	//Run: func(cmd *cobra.Command, args []string) {
	//	fmt.Println("Zaohua called")
	//},
	Run: zaohuaRunner,
}

func init() {
	rootCmd.AddCommand(ZaohuaCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// ZaohuaCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// ZaohuaCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func initZaohuaOptions() error{

	return nil
}

func zaohuaRunner(cmd *cobra.Command, args []string) {
	initZaohuaOptions()

	initial.InitialAggrator(consts.ZaohuaCmdRole)
}