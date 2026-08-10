package runtime

import "context"

// Cmd is a cancellable asynchronous command. A nil return value is ignored.
type Cmd func(context.Context) Msg
