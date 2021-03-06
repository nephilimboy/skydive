/*
 * Copyright (C) 2016 Red Hat, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy ofthe License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specificlanguage governing permissions and
 * limitations under the License.
 *
 */

package pod

import (
	"github.com/skydive-project/skydive/graffiti/graph"
	gws "github.com/skydive-project/skydive/graffiti/websocket"
	"github.com/skydive-project/skydive/logging"
	ws "github.com/skydive-project/skydive/websocket"
)

// TopologyForwarder forwards the topology to only one master server.
// When switching from one analyzer to another one the agent does a full
// re-sync since some messages could have been lost.
type TopologyForwarder struct {
	masterElection *ws.MasterElection
	graph          *graph.Graph
	host           string
}

func (t *TopologyForwarder) triggerResync() {
	logging.GetLogger().Infof("Start a re-sync for %s", t.host)

	t.graph.RLock()
	defer t.graph.RUnlock()

	// re-add all the nodes and edges
	msg := &gws.SyncMsg{
		Elements: t.graph.Elements(),
	}
	t.masterElection.SendMessageToMaster(gws.NewStructMessage(gws.SyncMsgType, msg))
}

// OnNewMaster is called by the master election mechanism when a new master is elected. In
// such case a "Re-sync" is triggered in order to be in sync with the new master.
func (t *TopologyForwarder) OnNewMaster(c ws.Speaker) {
	if c == nil {
		logging.GetLogger().Warn("Lost connection to master")
	} else {
		addr, port := c.GetAddrPort()
		logging.GetLogger().Infof("Using %s:%d as master of topology forwarder", addr, port)

		t.triggerResync()
	}
}

// OnNodeUpdated graph node updated event. Implements the EventListener interface.
func (t *TopologyForwarder) OnNodeUpdated(n *graph.Node) {
	t.masterElection.SendMessageToMaster(gws.NewStructMessage(gws.NodeUpdatedMsgType, n))
}

// OnNodeAdded graph node added event. Implements the EventListener interface.
func (t *TopologyForwarder) OnNodeAdded(n *graph.Node) {
	t.masterElection.SendMessageToMaster(gws.NewStructMessage(gws.NodeAddedMsgType, n))
}

// OnNodeDeleted graph node deleted event. Implements the EventListener interface.
func (t *TopologyForwarder) OnNodeDeleted(n *graph.Node) {
	t.masterElection.SendMessageToMaster(gws.NewStructMessage(gws.NodeDeletedMsgType, n))
}

// OnEdgeUpdated graph edge updated event. Implements the EventListener interface.
func (t *TopologyForwarder) OnEdgeUpdated(e *graph.Edge) {
	t.masterElection.SendMessageToMaster(gws.NewStructMessage(gws.EdgeUpdatedMsgType, e))
}

// OnEdgeAdded graph edge added event. Implements the EventListener interface.
func (t *TopologyForwarder) OnEdgeAdded(e *graph.Edge) {
	t.masterElection.SendMessageToMaster(gws.NewStructMessage(gws.EdgeAddedMsgType, e))
}

// OnEdgeDeleted graph edge deleted event. Implements the EventListener interface.
func (t *TopologyForwarder) OnEdgeDeleted(e *graph.Edge) {
	t.masterElection.SendMessageToMaster(gws.NewStructMessage(gws.EdgeDeletedMsgType, e))
}

// GetMaster returns the current analyzer the agent is sending its events to
func (t *TopologyForwarder) GetMaster() ws.Speaker {
	return t.masterElection.GetMaster()
}

// NewTopologyForwarder returns a new Graph forwarder which forwards event of the given graph
// to the given WebSocket JSON speakers.
func NewTopologyForwarder(host string, g *graph.Graph, pool ws.StructSpeakerPool) *TopologyForwarder {
	masterElection := ws.NewMasterElection(pool)

	t := &TopologyForwarder{
		masterElection: masterElection,
		graph:          g,
		host:           host,
	}

	masterElection.AddEventHandler(t)
	g.AddEventListener(t)

	return t
}
