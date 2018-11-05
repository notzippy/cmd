package parser

//
import (
	"go/ast"
	"github.com/revel/cmd/controller"
)

// Process the comment and return a list of functional annotations
func processComment(cg *ast.CommentGroup) (list controller.FunctionalAnnotations) {
	if cg == nil {
		return
	}
	// Extract the comments out look for ones with ampersands
	lines := extractComments(cg)
	for _,l := range lines {
		if a,e := parseAnnotation(l);e==nil {
			list = append(list,a)
		}
	}

	return
}

// Extract the comments to a string
func extractComments(commentGroup *ast.CommentGroup) []string {
	lines := make([]string, len(commentGroup.List))
	for i, comment := range commentGroup.List {
		lines[i] = comment.Text
	}
	return lines
}
