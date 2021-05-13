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
		cc.HandleTag(func(attrs AttributeList, content *ContentIterator) error {
			for _, a := range attrs {
				fmt.Printf(" %s=%s", a.Name, a.Value)
			}
			handleContent(content)
			return nil
		})
		fmt.Printf("\ndone\n")
	}
}

func handleContent(content *ContentIterator) {
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
			content.HandleTag(func(attrs AttributeList, chilcontent *ContentIterator) error {
				for _, a := range attrs {
					fmt.Printf(" %s=%s", a.Name, a.Value)
				}
				if content == nil {
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

const example01 = `<?xml version="1.0" encoding="UTF-8"?>
<!--comment-->
<root attr="val" attr2='val2'>
  <Reason>
    <id>1</id>
    <caption>Visual</caption>
    <type>1</type>
    <created>2019-04-09 14:18:21</created>
    <deleted>0</deleted>
    <position>1</position>
    <uid>5cace1ed-16b4-4598-b594-430bdc0e4b59</uid>
  </Reason>
  <Reason>
    <id>1</id>
    <unknown>blah</unknown>
    <caption>Another Reason</caption>
    <type>1</type>
    <created>2019-04-09 14:18:21</created>
    <deleted>0</deleted>
    <position>1</position>
    <uid>5cace1ed-16b4-4598-b594-430bdc0e4b59</uid>
  </Reason>
  <User>
    <id>1</id>
    <uid>d9d0d6a7-a8e2-4477-bb62-067f4152d1ff</uid>
    <name>admin</name>
    <image_index>1</image_index>
    <created>2019-04-09 09:40:53</created>
    <modified>2019-04-12 12:32:09</modified>
    <deleted>0</deleted>
    <login>admin</login>
    <pwd_hash_public>$2y$09$3UYSx/bJ5JX5L1EQjD2PuON8XFKowbMSbcP/j/VuJTkLVfnlrnWQW</pwd_hash_public>
    <pwd_hash_private>$2y$09$pK53BkSOitk95N3wWxg1kO</pwd_hash_private>
    <Ugroup>
      <id>3</id>
      <UgroupsUser>
        <id>1</id>
        <user_id>1</user_id>
        <ugroup_id>3</ugroup_id>
      </UgroupsUser>
    </Ugroup>
  </User>
  <Ugroup>
    <id>1</id>
    <name>standard</name>
    <sort_order>1</sort_order>
  </Ugroup>
  <Category>
    <id>1</id>
    <uid>c233cc1e-a00b-4b92-93d3-520c613ecd4a</uid>
    <name>Delta</name>
    <color>FF4040</color>
    <type>1</type>
  </Category>
  <Category>
    <id>2</id>
    <uid>5cacbb17-5c4c-4ead-8207-d198dc0e4b59</uid>
    <name>3T</name>
    <color>40F040</color>
    <type>1</type>
  </Category>
</root>
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
