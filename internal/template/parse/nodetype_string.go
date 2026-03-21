package parse

import "strconv"

func (i NodeType) String() string {
	// Upstream HTML5 node types (0-7).
	switch i {
	case ErrorNode:
		return "ErrorNode"
	case TextNode:
		return "TextNode"
	case DocumentNode:
		return "DocumentNode"
	case ElementNode:
		return "ElementNode"
	case CommentNode:
		return "CommentNode"
	case DoctypeNode:
		return "DoctypeNode"
	case RawNode:
		return "RawNode"
	case scopeMarkerNode:
		return "scopeMarkerNode"
	// Template node types (100+).
	case ExpressionNode:
		return "ExpressionNode"
	case IfNode:
		return "IfNode"
	case ElseNode:
		return "ElseNode"
	case EachNode:
		return "EachNode"
	case SnippetNode:
		return "SnippetNode"
	case ConstNode:
		return "ConstNode"
	case ifCloseNode:
		return "ifCloseNode"
	case eachCloseNode:
		return "eachCloseNode"
	case snippetCloseNode:
		return "snippetCloseNode"
	default:
		return "NodeType(" + strconv.FormatInt(int64(i), 10) + ")"
	}
}
