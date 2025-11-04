/*
Copyright © november 2025 vetab60 <al9xgr99n@gmail.com>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"PackageManager/internal"
	"PackageManager/internal/configs"
	"PackageManager/internal/storage"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/ztrue/tracerr"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "PackageManager",
	Short: "Remote client for remote storage",
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

var fromEnv *bool
var cfgFile *string

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	fromEnv = rootCmd.PersistentFlags().BoolP("env", "e", false, "read configs from environment")
	cfgFile = rootCmd.PersistentFlags().StringP("cfg", "f", "", "configs file (default is empty)")
	var pack = rootCmd.PersistentFlags().StringP("pack", "p", "packet.json", "input packet.json")
	var unpack = rootCmd.PersistentFlags().StringP("unpack", "u", "packet.json", "input packages.json")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	storage_path := rootCmd.PersistentFlags().StringP("storage_path", "s", ".", "path in remote storage server for saving files")
	output := rootCmd.PersistentFlags().StringP("output", "o", ".", "path for save fetching packages")

	viper.Set("pack", pack)
	viper.Set("unpack", unpack)
	viper.Set("storage_path", storage_path)
	viper.Set("output", output)
}

// initConfig reads in configs file and ENV variables if set.
func initConfig() {
	sshConfig := configs.NewSSHConfig()
	if *cfgFile != "" && *fromEnv {
		cobra.CheckErr(tracerr.New("cant use configs from environment and cfg file together, use onl one flag"))
	}
	if *cfgFile != "" {
		// Use configs file from the flag.
		viper.SetConfigFile(*cfgFile)
	} else if *fromEnv {
		viper.SetEnvPrefix("uploader")
		viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
		viper.AutomaticEnv() // read in environment variables that match

		err := sshConfig.LoadFromEnv()
		cobra.CheckErr(err)
		err = sshConfig.Validate()
		cobra.CheckErr(err)
		viper.Set("ssh-config", sshConfig)
		return
	} else {
		cobra.CheckErr("Config file not set")
	}

	// If a configs file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using configs file:", viper.ConfigFileUsed())
	}

	if err := viper.Unmarshal(&sshConfig); err != nil {
		cobra.CheckErr(err)
	}
	viper.Set("ssh-config", sshConfig)

	ctx := context.WithValue(context.Background(), "ssh-config", sshConfig)

	ctx = context.WithValue(ctx, "workerNum", 1)

	sshClient, err := storage.NewSshClient(ctx)
	if err != nil {
		log.Println(tracerr.Sprint(err))
	}

	rClient, err := internal.NewRemoteClient(ctx, sshClient)
	if err != nil {
		log.Println(tracerr.Sprint(err))
	}

	viper.Set("remote-client", rClient)
}
