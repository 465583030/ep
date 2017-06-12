package ep

import (
    "context"
)

// Scatter returns a distribute Runner that scatters its input uniformly to
// all other nodes such that the received datasets are dispatched in a round-
// robin to the nodes.
func Scatter() Runner {
    return nil
}

// Gather returns a distribute Runner that gathers all of its input into a
// single node. In all other nodes it will produce no output, but on the main
// node it will be passthrough from all of the other nodes
func Gather() Runner {
    return nil
}

// Broadcast returns a distribute Runner that duplicates its input to all
// other nodes. The output will be effectively a union of all of the inputs from
// all nodes (order not guaranteed)
func Broadcast() Runner {
    return nil
}

// Partition returns a distribute Runner that scatters its input to all other
// nodes, except that it uses the last Data column in the input datasets to
// determine the target node of each dataset. This is useful for partitioning
// based on the values in the data, thus guaranteeing that all equal values land
// in the same target node
func Partition() Runner {
    return nil
}

// Distribute a Runner to run on multiple nodes concurrently. The returned
// Runner acts as a Gather, in that it collects all of the values returned from
// the individual nodes. However, input to this node is not distributed - an
// explicit Scatter must exist for the input from the local node to be sent over
// to the other nodes.
func Distribute(runner Runner, nodes ...Node) (Runner, error) {
    uuid := uuid()
    for _, n := range nodes {
        conn, err := n.Connect(uuid)

        conn.Encode()
    }


    return nil, nil
}

// distribute is a Runner that exchanges data between peer nodes
type distribute struct {
    UUID string
}

func (d *distribute) Run(ctx context.Context, inp, out chan Dataset) (err error) {
    sources, targets, err := d.Connect(ctx)
    defer func() { append(sources, targets...).Close(err) }()
    if err != nil {
        return
    }

    // send the local data to the distributed target nodes
    go func() {
        for data := range inp {
            err = targets.Send(data)
            if err != nil {
                return
            }
        }
    }()

    // listen to all nodes for incoming data
    var data Dataset
    for {
        data, err = sources.Receive()
        if err != nil {
            return
        }

        out <- data
    }
}

func (d *distribute) Connect(ctx context.Context) (sources, targets conns, err error) {
    var isThisTarget bool // is this node also a destination?

    allNodes := ctx.Value("ep.AllNodes").([]Node)
    thisNode := ctx.Value("ep.ThisNode").(Node)
    masterNode := ctx.Value("ep.MasterNode").(Node)

    targetNodes = allNodes
    if d.send == sendGather {
        targetNodes = []Node{masterNode}
    }

    // open a connection to all target nodes
    for _, n := range targetNodes {
        isThisTarget = isThisTarget || n == thisNode // TODO: short-circuit
        conn, err = connect(n, d.UUID)
        if err != nil {
            return
        }

        targets = append(targets, conn)
    }

    // if we're also a destination, listen to all nodes
    for i := 0; isThisTarget && i < len(allNodes); i += 1 {
        n := AllNodes[i]

        // if we already established a connection to this node from the targets,
        // re-use it. We don't need 2 uni-directional connections.
        conn = targets.Get(n)
        if conn == nil {
            conn, err = connect(n, d.UUID)
        }

        if err != nil {
            return
        }

        sources = append(sources, conn)
    }
}
