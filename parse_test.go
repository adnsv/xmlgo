package xg

import (
	"fmt"
	"log"
	"testing"
)

func TestParse(t *testing.T) {
	//ShowTokens(example2)
	ParseExample(example01)
}

func ParseExample(buf string) {
	cc := Open(buf)
	if cc.NextTag() {
		fmt.Printf("root: %s", cc.Name())
		cc.HandleTag(func(attrs AttributeList, content *Content) error {
			for _, a := range attrs {
				fmt.Printf(" %s=%s", a.Name, a.Value)
			}
			handleContent(content)
			return nil
		})
		fmt.Printf("\ndone\n")
	}
}

func handleContent(content *Content) {
	for content.Next() {
		switch content.Kind() {
		case XmlDecl:
			fmt.Printf(`<?xml version="1.0" encoding="UTF-8"?>`)
		case Tag:
			n := content.Name()
			if n == "id" || n == "caption" {
				v := content.ChildStringContent()
				fmt.Printf("- %s=%s", n, v)
				continue
			}
			fmt.Printf("<%s", n)
			content.HandleTag(func(attrs AttributeList, chilcontent *Content) error {
				for _, a := range attrs {
					fmt.Printf(" %s=%s", a.Name, a.Value)
				}
				if chilcontent == nil {
					fmt.Printf("/>")
				} else {
					fmt.Print(">")
					handleContent(chilcontent)
					fmt.Printf("</%s>", n)
				}
				return nil
			})
		case SData:
			fmt.Printf("%s", content.Value())
		case CData:
			fmt.Printf("<![CDATA[%s]]>", content.Value())
		case Comment:
			fmt.Printf("<!--%s-->", content.Value())
		case PI:
			fmt.Printf("<?%s %s?>", content.Name(), content.Value())
		default:
			fmt.Printf("<unknown>")
		}
	}
}

func ShowTokens(buf string) {
	err := ParseTokens(buf, func(t *Token) error {
		switch t.Kind {
		case EOF:
			fmt.Printf("\n[EOF]\n")
			return nil
		case Err:
			return t.Error
		case XmlDecl:
			fmt.Printf("XMLDECL[%s]", t.Raw)
		case Tag:
			fmt.Printf("%s<TAG:%s", t.WhitePrefix, t.Name)
		case BeginContent:
			fmt.Printf("[")
		case CloseEmptyTag:
			fmt.Printf(">")
		case EndContent:
			fmt.Printf("]>")
		case Attrib:
			fmt.Printf(" %s=%s", t.Name, t.Value)
		case SData:
			fmt.Printf("%s", t.Value)
		case CData:
			fmt.Printf("<CDATA:[%s]>", t.Value)
		case Comment:
			fmt.Printf("%s<COMMENT:%s>", t.WhitePrefix, t.Value)
		case PI:
			fmt.Printf("%s<PI:%s %s>", t.WhitePrefix, t.Name, t.Value)
		}
		return nil
	})
	fmt.Printf("\n")
	if err != nil {
		log.Fatal(err)
	}
}

const example01 = `<?xml version="1.0" encoding="utf-8"?>
<!-- Generator: Adobe Illustrator 25.2.1, SVG Export Plug-In . SVG Version: 6.00 Build 0)  -->
<svg version="1.1" id="Layer_1" xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" x="0px" y="0px"
	 width="375.6px" height="233.2px" viewBox="0 0 375.6 233.2" style="enable-background:new 0 0 375.6 233.2;" xml:space="preserve"
	>
<style type="text/css">
	.st0{fill:none;stroke:#000000;stroke-width:0.5;stroke-linecap:round;stroke-linejoin:round;stroke-miterlimit:10;}
	.st1{fill:none;stroke:#000000;stroke-width:0.25;stroke-linecap:round;stroke-linejoin:round;stroke-miterlimit:10;}
	.st2{fill:#304499;stroke:#FFFFFF;stroke-width:0.5;stroke-miterlimit:10;}
	.st3{fill:#FFFFFF;stroke:#304499;stroke-width:0.3502;stroke-miterlimit:10;}
	.st4{fill:#010101;}
	.st5{font-family:'ArialMT';}
	.st6{font-size:8px;}
</style>
<path class="st0" d="M325.7,53.3c0.7-2.8,2.6-4.9,5.2-5.4"/>
<line class="st0" x1="317.1" y1="137.6" x2="341" y2="72.6"/>
<path class="st0" d="M327.3,153.5c0.2-0.5,0.1-1-0.2-1.5"/>
<line class="st0" x1="53.6" y1="168.3" x2="51.9" y2="168"/>
<path class="st1" d="M105,12.3c1.2,1.9,1.4,4.1,0.6,6.1"/>
<line class="st1" x1="18.6" y1="179.9" x2="27.6" y2="191.9"/>
<text transform="matrix(1 0 0 1 349.3292 195.6429)" class="st4 st5 st6">2</text>
</svg>
`
const example02 = `<?xml version="1.0" encoding="UTF-8"?>
<DocumentElement param="value">
	<!-- comment -->
	<FirstElement>
		&#xb6; Some Text
	</FirstElement>
	<?some_pi some_attr="some_value"?>
	<SecondElement param2="something">
		Pre-Text <Inline>Inlined text</Inline> Post-text.
	</SecondElement>
</DocumentElement>
<?another_pi some_attr="some_value"?>
`

const example03 = `<?xml version="1.0" encoding="utf-8"?>
<!-- Generator: Adobe Illustrator 15.0.0, SVG Export Plug-In . SVG Version: 6.00 Build 0) -->
<!DOCTYPE svg PUBLIC "-//W3C//DTD SVG 1.1 Tiny//EN" "http://www.w3.org/Graphics/SVG/1.1/DTD/svg11-tiny.dtd">
<svg version="1.1" baseProfile="tiny" id="Layer_1" xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" x="0px" y="0px" width="25.2px" height="25.2px" viewBox="0 0 25.2 25.2" xml:space="preserve">
	<path fill="#231F20" d="M12.601,0.9c6.451,0,11.7,5.249,11.7,11.7c0,6.451-5.249,11.701-11.7,11.701S0.9,19.051,0.9,12.6
	C0.9,6.149,6.149,0.9,12.601,0.9 M12.601,0C5.642,0,0,5.641,0,12.6c0,6.958,5.642,12.6,12.601,12.6s12.6-5.642,12.6-12.6
	C25.2,5.641,19.56,0,12.601,0L12.601,0z" />
	<polygon points="19.801,11.16 19.801,14.04 9.9,14.04 9.9,16.2 4.5,12.6 9.9,9 9.9,11.16 " />
</svg>`
