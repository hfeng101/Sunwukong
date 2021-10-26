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
	"flag"
	"github.com/hfeng101/Sunwukong/cmd/initial"
	"github.com/hfeng101/Sunwukong/util/consts"
	"github.com/spf13/cobra"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"time"
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

var (
	//XianqiName string
	//XianqiNamespace string
	//HoumaoName string
	//HoumaoNamespace string

	CpuInitializationPeriod time.Duration
	InitialReadinessDelay time.Duration
	DownScaleStabilizationWindow time.Duration
	ScaleTolerance float64
)

// 施法的后续造化流程
func init() {
	shifaCmd.AddCommand(ZaohuaCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// ZaohuaCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// ZaohuaCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func initZaohuaOptions() {
	//flag.StringVar(&XianqiName, "xianqi", "", "xianqi name")
	//flag.StringVar(&XianqiNamespace, "xianqiNamespace", "", "xianqi namespace")
	//flag.StringVar(&HoumaoName, "houmao", "", "houmao name")
	//flag.StringVar(&HoumaoNamespace, "houmaoNamespace", "", "houmao namespace")

	flag.DurationVar(&CpuInitializationPeriod, "cpu-initialization-period", consts.CpuInitializationPeriod,"The period after pod start when CPU samples might be skipped.")
	flag.DurationVar(&InitialReadinessDelay, "initial-readiness-delay", consts.InitialReadinessDelay,"The period after pod start during which readiness changes will be treated as initial readiness.")
	flag.DurationVar(&DownScaleStabilizationWindow, "down-scale-stabilization-window", 300, "The period for which autoscaler will look backwards and not scale down below any recommendation it made during that period." )
	flag.Float64Var(&ScaleTolerance, "scale-tolerance", consts.ScaleTolerance, "The minimum change (from 1.0) in desired-to-actual metrics ratio for the horizontal pod autoscaler to consider scaling.")

	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()
}

func zaohuaRunner(cmd *cobra.Command, args []string) {
	initZaohuaOptions()
	initParam := &initial.InitialParam{
		consts.ZaohuaCmdRole,
		CpuInitializationPeriod,
		InitialReadinessDelay,
		DownScaleStabilizationWindow,
		ScaleTolerance,
	}

	// 检测crd配置变更事件
	initial.InitialAggrator(initParam)
}
