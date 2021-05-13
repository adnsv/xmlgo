package xg

import (
	"fmt"
	"testing"
)

func TestParse(t *testing.T) {
	ShowTokens(exampleXML)
	ParseExample()
}

func ShowTokens(buf string) {
	tt := tokenizer{buf: buf}
	for {
		t := tt.Next()
		switch t.Kind {
		case StartToken:
			fmt.Printf("SOF\n")
		case DoneToken:
			fmt.Printf("\n[EOF]\n")
			return
		case ErrToken:
			fmt.Printf("\nERR: %s\n", t.EC.String())
			return
		case XmlDeclToken:
			fmt.Printf("XMLDECL[%s]", t.Raw)
		case OTagToken:
			fmt.Printf("%s<TAG:%s", t.WhitePrefix, t.Name)
		case ChildrenToken:
			fmt.Printf("[")
		case ETagToken:
			fmt.Printf(">")
		case CTagToken:
			fmt.Printf("]>")
		case AttribToken:
			fmt.Printf(" %s=%s", t.Name, t.Str)
		case SDataToken:
			fmt.Printf("%s", t.Str)
		case CDataToken:
			fmt.Printf("<CDATA:[%s]>", t.Str)
		case CommentToken:
			fmt.Printf("%s<COMMENT:%s>", t.WhitePrefix, t.Str)
		}
	}

}

func ParseExample() {
	Parse(exampleXML, tagPrinter)
}

func tagPrinter(tag NameString, raw string) (TagHandler, error) {
	fmt.Printf("<%s", tag)
	h := TagHandler{}
	h.OnAttr = func(n NameString, v RawString, raw string) error {
		//fmt.Printf(" %s=%s", n, v)
		fmt.Printf("%s", raw)
		return nil
	}
	h.OnStartContent = func(raw string) error {
		fmt.Printf(">")
		return nil
	}
	h.OnChildSD = func(v RawString, raw string) error {
		fmt.Printf("%s", v)
		return nil
	}
	h.OnChildTag = tagPrinter
	h.OnClose = func(empty bool, raw string) error {
		fmt.Printf("%s", raw)
		return nil
	}

	return h, nil
}

const exampleXML = `<?xml version="1.0" encoding="UTF-8"?>
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
