package draw2dfont

import (
	_ "embed"

	"github.com/golang/freetype/truetype"
	"github.com/llgcode/draw2d"
)

//go:embed luxisr.ttf
var luxisrTTF []byte

func init() {
	font, err := truetype.Parse(luxisrTTF)
	if err != nil {
		panic("failed to parse bundled font: " + err.Error())
	}
	fontData := draw2d.FontData{Name: "luxisr", Family: draw2d.FontFamilySans, Style: draw2d.FontStyleNormal}
	draw2d.RegisterFont(fontData, font)
	
	fontDataSans := draw2d.FontData{Name: "sans", Family: draw2d.FontFamilySans, Style: draw2d.FontStyleNormal}
	draw2d.RegisterFont(fontDataSans, font)
}
