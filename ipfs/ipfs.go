/*
 * IPFS is a global, versioned, peer-to-peer filesystem
 * CurrentVersionNumber is the current application's version literal
 *
 *
 */



/*

Handle starts handling the given signals, and will call the handler callback function each time a signal is caught. The function is passed the number of times the handler has been triggered in total, as well as the handler itself, so that the handling logic can use the handler's wait group to ensure clean shutdown when Close() is called.

*/

type IntrHandler struct {
    // contains filtered or unexported fields
}
