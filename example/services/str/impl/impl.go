package impl

import (
	"context"
	"strings"

	"github.com/tiki/example/svcdef"
)

type StrSrv struct{}

func (srv StrSrv) Reverse(ctx context.Context, msg *svcdef.StringMsg) (*svcdef.StringMsg, error) {
	s := msg.Str
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	s = string(runes)
	return &svcdef.StringMsg{
		Str: s,
	}, nil
}

func (srv StrSrv) UpperCase(ctx context.Context, msg *svcdef.StringMsg) (*svcdef.StringMsg, error) {
	return &svcdef.StringMsg{
		Str: strings.ToUpper(msg.Str),
	}, nil
}
