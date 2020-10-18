package transmission

// Success indicates result of success
const Success = "success"

// XSRFToken is the header name for the XSRF token
const XSRFToken = "X-Transmission-Session-Id"

// RPCPath is the url path for rpc
const RPCPath = "/transmission/rpc"

// ContentType is the content type for responses
const ContentType = "text/json; encoding=UTF-8"

const notImplemented = "Not Implemented"

const idRecentlyActive = "recently-active"

const tr_Status_Stopped = 0
const tr_Status_CheckWait = 1
const tr_Status_Check = 2
const tr_Status_DownloadWait = 3
const tr_Status_Download = 4
const tr_Status_SeedWait = 5
const tr_Status_Seed = 6

const tr_Pri_Low = -1
const tr_Pri_Norm = 0
const tr_Pri_High = 1
