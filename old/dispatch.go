package xcommon

import (
	"errors"
)

var NothingToDispatchError = errors.New("No handler to dispatch")

// func Dispatch(ctx context.Context, confResult *xcommon.ConfigurationResult) error {
// 	handler := confResult.SubcommandHandler
// 	if handler == nil {
// 		pflag.CommandLine.FlagUsages()
// 		return NothingToDispatchError
// 	}
// 	return handler(ctx)
// }
