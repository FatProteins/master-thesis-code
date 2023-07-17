package constants

import (
	"github.com/FatProteins/master-thesis-code/network/protocol"
	"github.com/FatProteins/master-thesis-code/util"
	"google.golang.org/protobuf/proto"
)

var voteRequestReceivedSize = proto.Size(&protocol.VoteRequestReceived{})
var voteReceivedSize = proto.Size(&protocol.VoteReceived{})
var logEntryReplicatedSize = proto.Size(&protocol.LogEntryReplicated{})
var logEntryCommitedSize = proto.Size(&protocol.LogEntryCommitted{})
var followerSuspectedSize = proto.Size(&protocol.FollowerSuspected{})
var leaderSuspectedSize = proto.Size(&protocol.LeaderSuspected{})
var maxMessageSize = util.Max(
	voteRequestReceivedSize,
	voteReceivedSize,
	logEntryReplicatedSize,
	logEntryCommitedSize,
	followerSuspectedSize,
	leaderSuspectedSize,
)

func MaxMessageSize() int {
	return maxMessageSize
}
