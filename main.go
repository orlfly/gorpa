package main

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"log"
	"strings"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/debugger"
	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/cdproto/domdebugger"
	"github.com/chromedp/chromedp"
	"gocv.io/x/gocv"
)

func travelSubtree(pageUrl, of string, img *gocv.Mat, opts ...chromedp.QueryOption) chromedp.Tasks {
	var nodes []*cdp.Node
	return chromedp.Tasks{
		chromedp.EmulateViewport(1920, 2000),
		chromedp.Navigate(pageUrl),
		chromedp.Nodes(of, &nodes, opts...),
		// ask chromedp to populate the subtree of a node
		chromedp.ActionFunc(func(c context.Context) error {
			// depth -1 for the entire subtree
			// do your best to limit the size of the subtree
			return dom.RequestChildNodes(nodes[0].NodeID).WithDepth(-1).Do(c)
		}),

		chromedp.ActionFunc(func(c context.Context) error {
			debugger.Enable().Do(c)
			domdebugger.SetEventListenerBreakpoint("click").Do(c)
			chromedp.ListenTarget(c, func(ev interface{}) {
				switch ev := ev.(type) {
				case *debugger.EventPaused:
					go fmt.Printf("Event Exception, console time > %s \n", ev.Data)
					go debugger.SetSkipAllPauses(true).Do(c)
					go debugger.Resume().WithTerminateOnResume(false).Do(c)
					go debugger.SetSkipAllPauses(false).Do(c)
				}
			})
			return nil
		}),
		chromedp.Sleep(time.Second * 50),
	}
}

func printNodes(nodes []*cdp.Node, c context.Context, img *gocv.Mat) {
	for _, node := range nodes {

		box, err := dom.GetBoxModel().WithNodeID(node.NodeID).Do(c)
		if err == nil {
			if node.NodeName == "#text" {
				content := strings.Trim(node.NodeValue, " ")
				if len(content) > 0 {
					cbox := QuadToBox(box.Border)
					gocv.Rectangle(img, image.Rect(int(cbox[0]), int(cbox[1]), int(cbox[2]), int(cbox[3])), color.RGBA{0, 0, 255, 0}, 3)
					fmt.Printf("%s:%v\n", node.NodeValue, QuadToBox(box.Border))
				}
			}
			if node.NodeName == "IMG" {
				cbox := QuadToBox(box.Border)
				gocv.Rectangle(img, image.Rect(int(cbox[0]), int(cbox[1]), int(cbox[2]), int(cbox[3])), color.RGBA{0, 0, 255, 0}, 3)
				fmt.Printf("%s:%v\n", node.Attributes, QuadToBox(box.Border))
			}
		}
		if node.ChildNodeCount > 0 {
			printNodes(node.Children, c, img)
		}
	}
}
func QuadToBox(q dom.Quad) []float64 {
	xmin, ymin, xmax, ymax := q[0], q[1], q[0], q[1]
	for i := 2; i < len(q); i = i + 2 {
		if xmin > q[i] {
			xmin = q[i]
		}
		if ymin > q[i+1] {
			ymin = q[i+1]
		}
		if xmax < q[i] {
			xmax = q[i]
		}
		if ymax < q[i+1] {
			ymax = q[i+1]
		}
	}
	return []float64{xmin, ymin, xmax, ymax}
}

func main() {
	ctx, cancel := chromedp.NewExecAllocator(context.Background(), append(chromedp.DefaultExecAllocatorOptions[:], chromedp.Flag("headless", false))...)
	defer cancel()
	ctx, cancel = chromedp.NewContext(ctx)
	defer cancel()
	img := gocv.NewMatWithSize(2000, 1920, gocv.MatTypeCV8UC3)
	defer img.Close()
	//run task list
	err := chromedp.Run(ctx,
		travelSubtree("http://news.baidu.com/", "body", &img, chromedp.ByQuery),
	)
	if err != nil {
		log.Fatal(err)
	}

	gocv.IMWrite("test.jpg", img)

}
