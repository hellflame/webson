package webson

const DEFAULT_TIMEOUT = 10

const DEFAULT_MAGIC_KEY = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

const DEFAULT_CHUNK_SIZE = 4 * 1024

// 16bits for streams, 1bit for stream cancel, 15bits for stream id
const MAX_STREAMS_IN_THEORY = 1 << 15
const DEFAULT_MAX_STREAMS = 1024

const DEFAULT_COMPRESS_LEVEL = 1

// related to msg frame structure & stream id convertion
const streamBytes = 2

const DEFAULT_POOL_WAIT = 5
const DEFAUTL_RETRY_INTERVAL = 5
