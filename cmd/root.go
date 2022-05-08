/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"errors"
	"fmt"
	"github.com/signintech/gopdf"
	"github.com/spf13/cobra"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
)

func getImageRect(source string) (*gopdf.Rect, error) {
	f, err := os.Open(source)
	if err != nil {
		return nil, err
	}
	img, _, err := image.Decode(f)
	if err != nil {
		return nil, err
	}
	b := img.Bounds()
	return &gopdf.Rect{W: float64(b.Max.X), H: float64(b.Max.Y)}, nil
}

func isIn(file string, ets []string) bool {
	ext := strings.ToLower(path.Ext(file))
	for _, v := range ets {
		if ext == v {
			return true
		}
	}
	return false
}

func createSingleFile(dir string) error {
	dir, _ = filepath.Abs(dir)
	var name string = dir + ".pdf"
	var baseName string = "[" + filepath.Base(name) + "]"
	var index int = 1
	var pdf gopdf.GoPdf = gopdf.GoPdf{}
	var pageRect *gopdf.Rect
	var imageRect *gopdf.Rect
	var x float64
	var y float64

	fmt.Println(fmt.Sprintf("%s开始合并：%s", baseName, name))
	if !free {
		if pageWidth > 0 && pageHeight > 0 {
			pageRect = &gopdf.Rect{W: pageWidth, H: pageHeight}
		} else {
			switch size {
			case "A0":
				pageRect = gopdf.PageSizeA0
			case "A1":
				pageRect = gopdf.PageSizeA1
			case "A2":
				pageRect = gopdf.PageSizeA2
			case "A3":
				pageRect = gopdf.PageSizeA3
			case "A4":
				pageRect = gopdf.PageSizeA4
			default:
				return errors.New("不支持的页面尺寸，自定义页面尺寸请使用--width及--height")
			}
		}
		if landscape {
			pageRect = &gopdf.Rect{W: pageRect.H, H: pageRect.W}
		}
	}

	pdf.Start(gopdf.Config{Unit: gopdf.UnitPT})

	if err := filepath.Walk(dir, func(ph string, info fs.FileInfo, err error) error {
		if info == nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		d := filepath.Dir(ph)
		if d != dir && d != "." {
			fmt.Println(fmt.Sprintf("%s已忽略子文件夹文件：%s", baseName, ph))
			return nil
		}
		if !isIn(ph, []string{".png", ".jpeg", ".jpg"}) {
			fmt.Println(fmt.Sprintf("%s已忽略非法图片文件：%s", baseName, ph))
			return nil
		}
		fmt.Println(fmt.Sprintf("%s第%d页：%s", baseName, index, ph))
		index += 1

		imageRect, err = getImageRect(ph)
		if err != nil {
			return err
		}
		if free {
			pageRect = imageRect
			x = 0
			y = 0
		} else {
			iw := imageRect.W / imageRect.H * pageRect.H
			if iw > pageRect.W {
				imageRect.H = imageRect.H / imageRect.W * pageRect.W
				imageRect.W = pageRect.W
				x = 0
				y = (pageRect.H - imageRect.H) / 2
			} else {
				imageRect.H, imageRect.W = pageRect.H, iw
				x = (pageRect.W - imageRect.W) / 2
				y = 0
			}
		}
		pdf.AddPageWithOption(gopdf.PageOption{PageSize: pageRect})
		return pdf.Image(ph, x, y, imageRect)

	}); err != nil {
		return err
	}

	if err := pdf.WritePdf(name); err != nil {
		return err
	}
	fmt.Println(fmt.Sprintf("%s合并成功：%s\n", baseName, name))
	return nil
}

func createMultipleFile(dir string) error {
	dir, _ = filepath.Abs(dir)
	if err := filepath.Walk(dir, func(ph string, info fs.FileInfo, err error) error {
		if info == nil {
			return err
		}
		if info.IsDir() && ph != dir {
			return createSingleFile(ph)
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}

var rootCmd = &cobra.Command{
	Version: "1.0.0",
	Use:     "img2pdf [-w page_width -e page_height] [-s page_size] dir...",
	Short:   "图片转PDF",
	Long: `将单个或多个图片文件合并为一个PDF文件。使用前需要将待合并图片统一放置于同一文件夹中，并将图片按照期望的合并顺序重命名。
合并时默认按照字母顺序排列图片，推荐使用类似001.jpg、002.jpg、003.jpg的命名方式。
合并时程序会从指定文件夹中查找所有图片（不包含子文件夹中图片），然后按照命名顺序合并为一份PDF文件，文件名称为指定的文件夹名+“.pdf”，保存路径为指定文件夹所在目录。
仅支持PNG及JPEG两种图片格式，其他图片格式需自行格式转换后方可使用本程序进行合并。`,
	Args: cobra.MinimumNArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		for _, v := range args {
			if !batch {
				if err := createSingleFile(v); err != nil {
					return err
				}
			} else {
				if err := createMultipleFile(v); err != nil {
					return err
				}
			}
		}
		return nil
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

var batch bool
var free bool
var size string
var landscape bool
var pageWidth float64
var pageHeight float64

func init() {
	rootCmd.Flags().BoolVarP(&batch, "batch", "b", false, "批处理模式，程序更改为处理指定文件夹下所有的子文件夹，对其分别进行合并")
	rootCmd.Flags().BoolVarP(&free, "free", "f", false, "保持图片原始尺寸，不做修改。指定时忽略-swel")
	rootCmd.Flags().StringVarP(&size, "size", "s", "A4", "PDF页面尺寸A0~A4")
	rootCmd.Flags().BoolVarP(&landscape, "landscape", "l", false, "将页面布局改为横置")
	rootCmd.Flags().Float64VarP(&pageWidth, "width", "w", 0, "指定PDF文件的默认页面宽度，单位PT。需和-e同时使用，指定时忽略-s")
	rootCmd.Flags().Float64VarP(&pageHeight, "height", "e", 0, "指定PDF文件的默认页面高度，单位PT。需和-w同时使用，指定时忽略-s")
}
