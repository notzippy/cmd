# Revel command line tools

Provides the `revel` command, used to create and run Revel apps.

Examples

revel run github.com/revel/revel/samples/chatroom

revel build github.com/revel/revel/samples/chatroom chatroom

revel debug github.com/revel/revel/samples/chatroom
 # This may or may not help print your variables
(gdb) source ~/go/src/pkg/runtime/runtime-gdb.py
#Set breakpoint
(gdb) break github.com/revel/revel/samples/chat/app/controllers/app.go:19
Breakpoint 1 at 0x499240: file ~/gopath/src/github.com/revel/revel/samples/chat/app/controllers/app.go, line 19.
(gdb)run
...
Breakpoint 1, github.com/revel/revel/samples/chat/app/controllers.Application.EnterDemo (c=..., user=..., demo=..., ~r2=...)
    at ~/gopath/src/github.com/revel/revel/samples/chat/app/controllers/app.go:19
19              if c.Validation.HasErrors() {
	
}
(gdb) info args
c = {*github.com/revel/revel.Controller = 0xc210096790}
user = 0xc2100e299f "assaa"
demo = 0xc2100e29aa "websocket"
~anon2 = {tab = 0x7ffff5f07b88, data = 0x7ffff5f07bc0}
