package executer

import "strings"

type CommnadContainer struct {
	Commands []map[string]interface{}
}

func (c *CommnadContainer) AddCommand(item string, kwargs map[string]interface{}) int {
	_kwargs := make(map[string]interface{})
	for k, v := range kwargs {
		_kwargs[strings.TrimLeft(k, "_")] = v
	}
	idx := len(c.Commands)
	c.Commands = append(c.Commands, _kwargs)
	return idx
}
