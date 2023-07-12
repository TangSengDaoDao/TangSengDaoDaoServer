package markdown

import (
	"fmt"
	"testing"
)

func TestToHtml(t *testing.T) {

	htm := ToHtml("a\n```go\n /** test **/ func Test(v []byte) (error){ fmt.Println(\"zdsdsdsd\")}\n```\nb `测试`")

	fmt.Println("htm--->", htm)

}
