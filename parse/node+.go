// This file is only intended to remove linter error in node.go
// Normally, public method should have comment, by replacing the return type by a non public alias,
// that remove the linter requirement to add a public comment.

package parse

type pos = Pos
type actionNode = *ActionNode
type boolNode = *BoolNode
type branchNode = *BranchNode
type chainNode = *ChainNode
type commandNode = *CommandNode
type dotNode = *DotNode
type fieldNode = *FieldNode
type identifierNode = *IdentifierNode
type ifNode = *IfNode
type listNode = *ListNode
type nilNode = *NilNode
type numberNode = *NumberNode
type pipeNode = *PipeNode
type rangeNode = *RangeNode
type stringNode = *StringNode
type templateNode = *TemplateNode
type textNode = *TextNode
type variableNode = *VariableNode
type withNode = *WithNode
