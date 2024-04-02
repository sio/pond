package server

import "fmt"

// NBD protocol description:
//	https://github.com/NetworkBlockDevice/nbd/blob/f8d7d3dbf1ef2ef84c92fe375ebc8674a79e25c2/doc/proto.md

const (
	NBDMAGIC uint64 = 0x4e42444d41474943
	IHAVEOPT uint64 = 0x49484156454F5054
)

type handshakeFlag uint16

func (f handshakeFlag) String() string {
	return fmt.Sprintf("NBD handshake flag: %016b", f)
}

const (
	NBD_FLAG_FIXED_NEWSTYLE handshakeFlag = 1 << 0
	NBD_FLAG_NO_ZEROES      handshakeFlag = 1 << 1
)

type transmissionFlag uint16

func (f transmissionFlag) String() string {
	return fmt.Sprintf("NBD transmission flag: %016b", f)
}

const (
	NBD_FLAG_HAS_FLAGS      transmissionFlag = 1 << 0
	NBD_FLAG_READ_ONLY      transmissionFlag = 1 << 1
	NBD_FLAG_CAN_MULTI_CONN transmissionFlag = 1 << 8
	NBD_FLAG_SEND_CACHE     transmissionFlag = 1 << 10
)

type optionType uint32

func (o optionType) String() string {
	return fmt.Sprintf("NBD option: %d", o)
}

const (
	NBD_OPT_EXPORT_NAME      optionType = 1
	NBD_OPT_ABORT            optionType = 2
	NBD_OPT_LIST             optionType = 3
	NBD_OPT_STARTTLS         optionType = 5
	NBD_OPT_INFO             optionType = 6
	NBD_OPT_GO               optionType = 7
	NBD_OPT_STRUCTURED_REPLY optionType = 8
)

type optionReply uint32

func (o optionReply) String() string {
	if o < nbd_rep_error {
		return fmt.Sprintf("NBD option reply: %d", o)
	} else {
		return fmt.Sprintf("NBD option reply error: %d", o-nbd_rep_error)
	}
}

const (
	NBD_REP_ACK                 optionReply = 1
	NBD_REP_SERVER              optionReply = 2
	NBD_REP_INFO                optionReply = 3
	nbd_rep_error               optionReply = (1 << 31)
	NBD_REP_ERR_UNSUP           optionReply = nbd_rep_error + 1
	NBD_REP_ERR_POLICY          optionReply = nbd_rep_error + 2
	NBD_REP_ERR_INVALID         optionReply = nbd_rep_error + 3
	NBD_REP_ERR_PLATFORM        optionReply = nbd_rep_error + 4
	NBD_REP_ERR_TLS_REQD        optionReply = nbd_rep_error + 5
	NBD_REP_ERR_UNKNOWN         optionReply = nbd_rep_error + 6
	NBD_REP_ERR_SHUTDOWN        optionReply = nbd_rep_error + 7
	NBD_REP_ERR_BLOCK_SIZE_REQD optionReply = nbd_rep_error + 8
	NBD_REP_ERR_TOO_BIG         optionReply = nbd_rep_error + 9
	NBD_REP_ERR_EXT_HEADER_REQD optionReply = nbd_rep_error + 10
)

type infoType uint16

func (i infoType) String() string {
	return fmt.Sprintf("NBD export info (type %d)", i)
}

const (
	NBD_INFO_EXPORT      infoType = 0
	NBD_INFO_NAME        infoType = 1
	NBD_INFO_DESCRIPTION infoType = 2
	NBD_INFO_BLOCK_SIZE  infoType = 3
)

type replyFlag uint16

func (f replyFlag) String() string {
	return fmt.Sprintf("NBD reply flag: %016b", f)
}

const (
	NBD_REPLY_FLAG_DONE replyFlag = 1 << 0
)

type replyType uint16

func (r replyType) String() string {
	return fmt.Sprintf("NBD reply type: %d", r)
}

const (
	NBD_REPLY_TYPE_NONE         replyType = 0
	NBD_REPLY_TYPE_OFFSET_DATA  replyType = 1
	NBD_REPLY_TYPE_OFFSET_HOLE  replyType = 2
	NBD_REPLY_TYPE_ERROR        replyType = (1 << 15) + 1
	NBD_REPLY_TYPE_ERROR_OFFSET replyType = (1 << 15) + 2
)

type requestType uint16

func (r requestType) String() string {
	return fmt.Sprintf("NBD request type: %d", r)
}

const (
	NBD_CMD_READ  requestType = 0
	NBD_CMD_DISC  requestType = 2
	NBD_CMD_CACHE requestType = 5
)

type nbdError uint32

func (e nbdError) Error() string {
	text, ok := nbdErrorText[e]
	if !ok {
		return fmt.Sprintf("NBD error %d", e)
	} else {
		return fmt.Sprintf("NBD error %d: %s", e, text)
	}
}

func (e nbdError) String() string {
	return e.Error()
}

const (
	NBD_EPERM     nbdError = 1
	NBD_EIO       nbdError = 5
	NBD_ENOMEM    nbdError = 12
	NBD_EINVAL    nbdError = 22
	NBD_ENOSPC    nbdError = 28
	NBD_EOVERFLOW nbdError = 75
	NBD_ENOTSUP   nbdError = 95
	NBD_ESHUTDOWN nbdError = 108
)

var nbdErrorText = map[nbdError]string{
	NBD_EPERM:     "Operation not permitted",
	NBD_EIO:       "Input/output error",
	NBD_ENOMEM:    "Cannot allocate memory",
	NBD_EINVAL:    "Invalid argument",
	NBD_ENOSPC:    "No space left on device",
	NBD_EOVERFLOW: "Value too large",
	NBD_ENOTSUP:   "Operation not supported",
	NBD_ESHUTDOWN: "Server is in the process of being shut down",
}
