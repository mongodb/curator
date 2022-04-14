package greenbay

import (
	"encoding/json"
	"fmt"

	"github.com/mongodb/grip/message"
)

type jsonOutput struct {
	output       CheckOutput
	message.Base `bson:"-" json:"-" yaml:"-"`
}

func (o *jsonOutput) Raw() interface{} { return o.output }
func (o *jsonOutput) Loggable() bool   { return true }
func (o *jsonOutput) String() string {
	out, err := json.Marshal(o)
	if err != nil {
		return fmt.Sprintf("processing result for '%s' (%+v): %+v",
			o.output.Name, err, o.output)
	}

	return string(out)
}
