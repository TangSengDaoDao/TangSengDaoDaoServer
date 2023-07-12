package markdown

import (
	"fmt"
	"io"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/ast"
	"github.com/gomarkdown/markdown/html"
	"github.com/sourcegraph/syntaxhighlight"
)

func ToHtml(v string) string {
	if v == "" {
		return ""
	}
	htmlFlags := html.CommonFlags | html.HrefTargetBlank
	opts := html.RendererOptions{
		Flags:          htmlFlags,
		RenderNodeHook: renderHookCodeBlock,
	}
	renderer := html.NewRenderer(opts)

	return string(markdown.ToHTML([]byte(v), nil, renderer))
}

func renderHookCodeBlock(w io.Writer, node ast.Node, entering bool) (ast.WalkStatus, bool) {

	_, ok := node.(*ast.Code)
	if ok {
		fmt.Println("code-------------->")
		w.Write([]byte(fmt.Sprintf("<pre class=\"notranslate\">%s</pre>", string(node.AsLeaf().Literal))))
		return ast.GoToNext, true
	}

	_, ok = node.(*ast.CodeBlock)
	if ok {
		syncHtml, _ := syntaxhighlight.AsHTML(node.AsLeaf().Literal, func(options *syntaxhighlight.HTMLConfig) {
			options.Type = "pl-en"
			options.Keyword = "pl-k"
			options.Plaintext = "pl-s1"
			options.String = "pl-s1"
			options.Comment = "pl-c"
		})
		w.Write([]byte(fmt.Sprintf("<pre class=\"notranslate\">%s</pre>", string(syncHtml))))
		return ast.GoToNext, true
	}
	// test := syntaxhighlight.HTMLConfig{
	// 	String:        "str",
	// 	Keyword:       "kwd",
	// 	Comment:       "com",
	// 	Type:          "typ",
	// 	Literal:       "lit",
	// 	Punctuation:   "pun",
	// 	Plaintext:     "pln",
	// 	Tag:           "tag",
	// 	HTMLTag:       "htm",
	// 	HTMLAttrName:  "atn",
	// 	HTMLAttrValue: "atv",
	// 	Decimal:       "dec",
	// 	Whitespace:    "",
	// }
	return ast.GoToNext, false

}
