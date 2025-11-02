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
	"RemoteUploader/internal"
	"RemoteUploader/internal/models"
	"encoding/json"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// removeCmd represents the remove command
var removeCmd = &cobra.Command{
	Use:   "remove",
	Short: "remove exist package",
	Run: func(cmd *cobra.Command, args []string) {
		rClient, ok := viper.Get("remote-client").(*internal.RemoteClient)
		if !ok {
			cobra.CheckErr("remote-client is not a valid remote client")
		}
		err := rClient.Remove(models.Delete(getUnpack()))
		if err != nil {
			cobra.CheckErr(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(removeCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// removeCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// removeCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func getUnpack() models.Unpack {
	unpackFile, ok := viper.Get("unpack").(*string)
	if !ok {
		cobra.CheckErr("unpack is not a string")
	}
	stat, err := os.Stat(*unpackFile)
	if err != nil {
		cobra.CheckErr(err)
	}
	if stat.Size() > max_pack_file_size {
		cobra.CheckErr("unpackFile file size too large")
	}

	bs, err := os.ReadFile(*unpackFile)
	if err != nil {
		cobra.CheckErr(err)
	}

	unpack := models.Unpack{}

	err = json.Unmarshal(bs, &unpack)
	if err != nil {
		cobra.CheckErr(err)
	}
	if unpack.Packages == nil {
		cobra.CheckErr("unpack package is empty")
	}
	return unpack
}
