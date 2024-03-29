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
  "fmt"
  "github.com/hfeng101/Sunwukong/cmd/initial"
  "github.com/hfeng101/Sunwukong/util/consts"
  "github.com/spf13/cobra"
  "os"

  homedir "github.com/mitchellh/go-homedir"
  "github.com/spf13/viper"
)


var cfgFile string


// shifaCmd represents the base command when called without any subcommands
var shifaCmd = &cobra.Command{
  Use:   "Sunwukong",
  Short: "A brief description of your application",
  Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
  // Uncomment the following line if your bare application
  // has an action associated with it:
  //	Run: func(cmd *cobra.Command, args []string) { },
  Run: shifaRunner,
}

// Execute adds all child commands to the shifa command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the shifaCmd.
func Execute() {
  if err := shifaCmd.Execute(); err != nil {
    fmt.Println(err)
    os.Exit(1)
  }
}

func init() {
  cobra.OnInitialize(initConfig)

  // Here you will define your flags and configuration settings.
  // Cobra supports persistent flags, which, if defined here,
  // will be global for your application.

  shifaCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.Sunwukong.yaml)")


  // Cobra also supports local flags, which will only run
  // when this action is called directly.
  shifaCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}


// initConfig reads in config file and ENV variables if set.
func initConfig() {
  if cfgFile != "" {
    // Use config file from the flag.
    viper.SetConfigFile(cfgFile)
  } else {
    // Find home directory.
    home, err := homedir.Dir()
    if err != nil {
      fmt.Println(err)
      os.Exit(1)
    }

    // Search config in home directory with name ".Sunwukong" (without extension).
    viper.AddConfigPath(home)
    viper.SetConfigName(".Sunwukong")
  }

  viper.AutomaticEnv() // read in environment variables that match

  // If a config file is found, read it in.
  if err := viper.ReadInConfig(); err == nil {
    fmt.Println("Using config file:", viper.ConfigFileUsed())
  }
}

func initshifaOptions() error{

  return nil
}

func shifaRunner(cmd *cobra.Command, args []string){
  initshifaOptions()

  initParam := &initial.InitialParam{
    Role: consts.ShifaCmdRole,
  }

  initial.InitialAggrator(initParam)
}

