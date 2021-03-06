package raft

//
// this is an outline of the API that raft must expose to
// the service (or tester). see comments below for
// each of these functions for more details.
//
// rf = Make(...)
//   create a new Raft server.
// rf.Start(command interface{}) (index, term, isleader)
//   start agreement on a new log entry
// rf.GetState() (term, isLeader)
//   ask a Raft for its current term, and whether it thinks it is leader
// ApplyMsg
//   each time a new entry is committed to the log, each Raft peer
//   should send an ApplyMsg to the service (or tester)
//   in the same server.
//

import (
	//	"bytes"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	//	"6.824/labgob"
	"6.824/labrpc"
)

//
// as each Raft peer becomes aware that successive log entries are
// committed, the peer should send an ApplyMsg to the service (or
// tester) on the same server, via the applyCh passed to Make(). set
// CommandValid to true to indicate that the ApplyMsg contains a newly
// committed log entry.
//
// in part 2D you'll want to send other kinds of messages (e.g.,
// snapshots) on the applyCh, but set CommandValid to false for these
// other uses.
//
type ApplyMsg struct {
	CommandValid bool
	Command      interface{}
	CommandIndex int

	// For 2D:
	SnapshotValid bool
	Snapshot      []byte
	SnapshotTerm  int
	SnapshotIndex int
}

const FOLLOWER int8 = 1
const CANDIDATE int8 = 2
const LEADER int8 = 3

//
// A Go object implementing a single Raft peer.
//
type Raft struct {
	mu        sync.Mutex          // Lock to protect shared access to this peer's state
	peers     []*labrpc.ClientEnd // RPC end points of all peers
	persister *Persister          // Object to hold this peer's persisted state
	me        int                 // this peer's index into peers[]
	dead      int32               // set by Kill()

	// server status
	status int8

	// for counting heartsbeats timeout(ms)
	beatTimeout       int
	lastBeatCheckTime int64

	// vote count
	voteCount int

	// Your data here (2A, 2B, 2C).
	// Look at the paper's Figure 2 for a description of what
	// state a Raft server must maintain.
	currentTerm int
	votedFor    int
}

// return currentTerm and whether this server
// believes it is the leader.
func (rf *Raft) GetState() (int, bool) {

	// Your code here (2A).
	return rf.currentTerm, rf.status == LEADER
}

//
// save Raft's persistent state to stable storage,
// where it can later be retrieved after a crash and restart.
// see paper's Figure 2 for a description of what should be persistent.
//
func (rf *Raft) persist() {
	// Your code here (2C).
	// Example:
	// w := new(bytes.Buffer)
	// e := labgob.NewEncoder(w)
	// e.Encode(rf.xxx)
	// e.Encode(rf.yyy)
	// data := w.Bytes()
	// rf.persister.SaveRaftState(data)
}

//
// restore previously persisted state.
//
func (rf *Raft) readPersist(data []byte) {
	if data == nil || len(data) < 1 { // bootstrap without any state?
		return
	}
	// Your code here (2C).
	// Example:
	// r := bytes.NewBuffer(data)
	// d := labgob.NewDecoder(r)
	// var xxx
	// var yyy
	// if d.Decode(&xxx) != nil ||
	//    d.Decode(&yyy) != nil {
	//   error...
	// } else {
	//   rf.xxx = xxx
	//   rf.yyy = yyy
	// }
}

//
// A service wants to switch to snapshot.  Only do so if Raft hasn't
// have more recent info since it communicate the snapshot on applyCh.
//
func (rf *Raft) CondInstallSnapshot(lastIncludedTerm int, lastIncludedIndex int, snapshot []byte) bool {

	// Your code here (2D).

	return true
}

// the service says it has created a snapshot that has
// all info up to and including index. this means the
// service no longer needs the log through (and including)
// that index. Raft should now trim its log as much as possible.
func (rf *Raft) Snapshot(index int, snapshot []byte) {
	// Your code here (2D).

}

//
// example RequestVote RPC arguments structure.
// field names must start with capital letters!
//
type RequestVoteArgs struct {
	// Your data here (2A, 2B).
	Term         int
	CandidateId  int
	LastLogIndex int
	LastLogTerm  int
}

//
// example RequestVote RPC reply structure.
// field names must start with capital letters!
//
type RequestVoteReply struct {
	// Your data here (2A).
	Term        int
	VoteGranted bool
}

//
// example RequestVote RPC handler.
//
func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) {
	// Your code here (2A, 2B).

	rf.mu.Lock()
	defer rf.mu.Unlock()

	fmt.Printf("[peer %d, currentTerm %d]: receive RequestVote from peer %d, current votedFor: %d\n", rf.me, rf.currentTerm, args.CandidateId, rf.votedFor)
	reply.Term = rf.currentTerm

	if args.Term < rf.currentTerm {
		reply.VoteGranted = false
		return
	}

	if args.Term > rf.currentTerm {
		rf.status = FOLLOWER
		rf.currentTerm = args.Term
		rf.votedFor = -1
	}

	if rf.votedFor == -1 || rf.votedFor == args.CandidateId {
		reply.VoteGranted = true
		rf.votedFor = args.CandidateId
	}
}

type AppendEntriesArg struct {
	Term         int
	LeaderId     int
	PrevLogIndex int
	PrevLogTerm  int
	Entries      []int
	LeaderCommit int
}

type AppendEntriesReply struct {
	Term    int
	Success bool
}

func (rf *Raft) AppendEntries(args *AppendEntriesArg, reply *AppendEntriesReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	reply.Term = rf.currentTerm

	if args.Term > rf.currentTerm {
		rf.status = FOLLOWER
		rf.currentTerm = args.Term
		rf.votedFor = -1
	}

	if args.Term < rf.currentTerm {
		reply.Success = false
	} else {
		reply.Success = true
		rf.lastBeatCheckTime = time.Now().UnixMilli()
	}
}

//
// example code to send a RequestVote RPC to a server.
// server is the index of the target server in rf.peers[].
// expects RPC arguments in args.
// fills in *reply with RPC reply, so caller should
// pass &reply.
// the types of the args and reply passed to Call() must be
// the same as the types of the arguments declared in the
// handler function (including whether they are pointers).
//
// The labrpc package simulates a lossy network, in which servers
// may be unreachable, and in which requests and replies may be lost.
// Call() sends a request and waits for a reply. If a reply arrives
// within a timeout interval, Call() returns true; otherwise
// Call() returns false. Thus Call() may not return for a while.
// A false return can be caused by a dead server, a live server that
// can't be reached, a lost request, or a lost reply.
//
// Call() is guaranteed to return (perhaps after a delay) *except* if the
// handler function on the server side does not return.  Thus there
// is no need to implement your own timeouts around Call().
//
// look at the comments in ../labrpc/labrpc.go for more details.
//
// if you're having trouble getting RPC to work, check that you've
// capitalized all field names in structs passed over RPC, and
// that the caller passes the address of the reply struct with &, not
// the struct itself.
//
func (rf *Raft) sendRequestVote(server int, args *RequestVoteArgs, reply *RequestVoteReply) bool {
	ok := rf.peers[server].Call("Raft.RequestVote", args, reply)
	return ok
}

func (rf *Raft) sendAppendEntries(server int, args *AppendEntriesArg, reply *AppendEntriesReply) bool {
	ok := rf.peers[server].Call("Raft.AppendEntries", args, reply)
	return ok
}

//
// the service using Raft (e.g. a k/v server) wants to start
// agreement on the next command to be appended to Raft's log. if this
// server isn't the leader, returns false. otherwise start the
// agreement and return immediately. there is no guarantee that this
// command will ever be committed to the Raft log, since the leader
// may fail or lose an election. even if the Raft instance has been killed,
// this function should return gracefully.
//
// the first return value is the index that the command will appear at
// if it's ever committed. the second return value is the current
// term. the third return value is true if this server believes it is
// the leader.
//
func (rf *Raft) Start(command interface{}) (int, int, bool) {
	index := -1
	term := -1
	isLeader := true

	// Your code here (2B).

	return index, term, isLeader
}

//
// the tester doesn't halt goroutines created by Raft after each test,
// but it does call the Kill() method. your code can use killed() to
// check whether Kill() has been called. the use of atomic avoids the
// need for a lock.
//
// the issue is that long-running goroutines use memory and may chew
// up CPU time, perhaps causing later tests to fail and generating
// confusing debug output. any goroutine with a long-running loop
// should call killed() to check whether it should stop.
//
func (rf *Raft) Kill() {
	atomic.StoreInt32(&rf.dead, 1)
	// Your code here, if desired.
}

func (rf *Raft) killed() bool {
	z := atomic.LoadInt32(&rf.dead)
	return z == 1
}

func (rf *Raft) sendVote(p int) {
	rf.mu.Lock()

	fmt.Printf("[peer %d, currentTerm %d]: require vote for peer %d\n", rf.me, rf.currentTerm, p)

	arg := RequestVoteArgs{Term: rf.currentTerm, CandidateId: rf.me}
	reply := RequestVoteReply{}

	rf.mu.Unlock()
	ok := rf.sendRequestVote(p, &arg, &reply)
	fmt.Printf("[peer %d, currentTerm %d]: get vote from peer %d with %v\n", rf.me, rf.currentTerm, p, reply.VoteGranted)
	if ok {
		rf.mu.Lock()
		defer rf.mu.Unlock()

		if rf.status != CANDIDATE {
			return
		}

		if reply.VoteGranted {
			rf.voteCount++

			if rf.voteCount >= int(math.Ceil(float64(len(rf.peers))/2)) {
				rf.status = LEADER
				fmt.Printf("[peer %d, currentTerm %d]: i'm leader now!\n", rf.me, rf.currentTerm)
			}
		} else if reply.Term > rf.currentTerm {
			rf.status = FOLLOWER
		}

		if rf.status == LEADER {
			go rf.heartbeat()
		}
	}
}

// election
func (rf *Raft) election() {
	fmt.Printf("[peer %d, currentTerm %d]: heartbeat timeout, start election\n", rf.me, rf.currentTerm)
	rf.mu.Lock()
	rf.status = CANDIDATE
	rf.voteCount = 1
	rf.votedFor = rf.me
	rf.currentTerm++
	rf.mu.Unlock()

	for i := range rf.peers {
		if i != rf.me {
			go rf.sendVote(i)
		}
	}
}

func (rf *Raft) sendBeat(server int) {
	// fmt.Printf("[peer %d, currentTerm %d]: send AppendEntries to peer %d\n", rf.me, rf.currentTerm, server)
	args := AppendEntriesArg{Term: rf.currentTerm, LeaderId: rf.me}
	reply := AppendEntriesReply{}

	if rf.status != LEADER {
		return
	}

	ok := rf.sendAppendEntries(server, &args, &reply)

	if ok {
		rf.mu.Lock()
		defer rf.mu.Unlock()
		if reply.Term > rf.currentTerm {
			rf.status = FOLLOWER
			rf.lastBeatCheckTime = time.Now().UnixMilli()
		}
	}
}

func (rf *Raft) heartbeat() {
	for {
		rf.mu.Lock()
		if rf.status == LEADER {
			rf.mu.Unlock()
			for i := range rf.peers {
				if i != rf.me {
					go rf.sendBeat(i)
				}
			}
			time.Sleep(100 * time.Millisecond)
		} else {
			rf.mu.Unlock()
			return
		}
	}
}

// The ticker go routine starts a new election if this peer hasn't received
// heartsbeats recently.
func (rf *Raft) ticker() {
	fmt.Printf("[peer %d, currentTerm %d]: started!\n", rf.me, rf.currentTerm)
	for rf.killed() == false {

		// Your code here to check if a leader election should
		// be started and to randomize sleeping time using
		// time.Sleep().
		rf.mu.Lock()

		if rf.status == LEADER {
			rf.mu.Unlock()
			time.Sleep(50 * time.Millisecond)
		} else {
			lastTime := rf.lastBeatCheckTime
			timeout := rf.beatTimeout
			rf.mu.Unlock()
			time.Sleep(time.Duration(timeout) * time.Millisecond)

			rf.mu.Lock()
			if rf.lastBeatCheckTime == lastTime {
				rf.beatTimeout = rand.Intn(200) + 300
				rf.mu.Unlock()
				rf.election()
			} else {
				rf.mu.Unlock()
			}
		}
	}
}

//
// the service or tester wants to create a Raft server. the ports
// of all the Raft servers (including this one) are in peers[]. this
// server's port is peers[me]. all the servers' peers[] arrays
// have the same order. persister is a place for this server to
// save its persistent state, and also initially holds the most
// recent saved state, if any. applyCh is a channel on which the
// tester or service expects Raft to send ApplyMsg messages.
// Make() must return quickly, so it should start goroutines
// for any long-running work.
//
func Make(peers []*labrpc.ClientEnd, me int,
	persister *Persister, applyCh chan ApplyMsg) *Raft {
	rf := &Raft{}
	rf.peers = peers
	rf.persister = persister
	rf.me = me

	// initialize status
	rf.status = FOLLOWER

	// Your initialization code here (2A, 2B, 2C).
	// initialize heartbeat info
	rf.beatTimeout = 300
	rf.lastBeatCheckTime = time.Now().UnixMilli()

	// initialize from state persisted before a crash
	rf.readPersist(persister.ReadRaftState())
	rf.votedFor = -1

	// start ticker goroutine to start elections
	go rf.ticker()

	return rf
}
