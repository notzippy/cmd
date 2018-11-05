package parser

import "github.com/revel/cmd/model"

type (
	// The route parser examines the
	routeParser struct {

	}
)

var RouteParser = &routeParser{}

// The router parser will analyze the route file, and map that information
// to the controllers Options. That way the server will only need to read
// information from one area and we can drop the route file
func (r *routeParser)Parse(paths model.RevelContainer, sourceModel *model.SourceInfo) {
	// Scan all the modules and fetch any route paths that may be configured


}