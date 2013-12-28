package main

import (
    "fmt";
    "net";
    "os";
    "container/list";
    "bytes";
    "flag";
)

type ClientChat struct {
    Name string;
    IN chan string;
    OUT chan string;
    Con net.Conn;
    Quit chan bool;
    ListChain *list.List;
}

func Log(v ...string) {
    ret := fmt.Sprint(v);
    fmt.Printf("SERVER: %s\n", ret);
}

func (c ClientChat) Read(buf []byte) bool{
    _, err := c.Con.Read(buf);
    if err!=nil {
        c.Close();
        return false;
    }
    Log("Read():  ", " bytes");
    return true;
}

func (c *ClientChat) Close() {
    c.Quit<-true;
    c.Con.Close();
    c.deleteFromList();
}

func (c *ClientChat) Equal(cl *ClientChat) bool {
    if bytes.EqualFold([]byte(c.Name), []byte(cl.Name)) {
        if c.Con == cl.Con {
            return true;
        }
    }
    return false;
}

func (c *ClientChat) deleteFromList() {
    for e := c.ListChain.Front(); e != nil; e = e.Next() {
        client := e.Value.(ClientChat);
        if c.Equal(&client) {
            Log("deleteFromList(): ", c.Name);
            c.ListChain.Remove(e);
        }
    }
}

func test(err error, mesg string) {
    if err!=nil {
        fmt.Printf("SERVER: ERROR: ", mesg);
         os.Exit(-1);
    } else {
        Log("Ok: ", mesg);
    }
}

func handlingINOUT(IN <-chan string, lst *list.List) {
    for {
        Log("handlingINOUT(): wait for input");
        input := <-IN;  // input, get from client
        // send to all client back
        Log("handlingINOUT(): handling input: ", input);
        for value := lst.Front(); value != nil; value = value.Next() {
            client := value.Value.(ClientChat)
            Log("handlingINOUT(): send to client: ", client.Name);
            client.IN<- input;
        }
    }
}


func clientreceiver(client *ClientChat) {
    buf := make([]byte, 2048);

    Log("clientreceiver(): start for: ", client.Name);
    for client.Read(buf) {

        if bytes.EqualFold(buf, []byte("/quit")) {
            client.Close();
            break;
        }
        Log("clientreceiver(): received from ",client.Name, " (", string(buf), ")");
        send := client.Name+"> "+string(buf);
        client.OUT<- send;
        for i:=0; i<2048;i++ {
            buf[i]=0x00;
        }
    }

    client.OUT <- client.Name+" has left chat";
    Log("clientreceiver(): stop for: ", client.Name);
}

func clientsender(client *ClientChat) {
    Log("clientsender(): start for: ", client.Name);
    for {
        Log("clientsender(): wait for input to send");
        select {
            case buf := <- client.IN:
                Log("clientsender(): send to \"", client.Name, "\": ", string(buf));
                client.Con.Write([]byte(buf));
            case <-client.Quit:
                Log("clientsender(): client want to quit");
                client.Con.Close();
                break;
        }
    }
    Log("clientsender(): stop for: ", client.Name);
}

func clientHandling(con net.Conn, ch chan string, lst *list.List) {
    buf := make([]byte, 1024);
    con.Read(buf);
    name := string(buf);
    newclient := &ClientChat{name, make(chan string), ch, con, make(chan bool), lst};

    Log("clientHandling(): for ", name);
    go clientsender(newclient);
    go clientreceiver(newclient);
    lst.PushBack(*newclient);
    ch<- name+" has joined the chat";
    ch<- "\nl33t stuff only";
}

func main() {
    flag.Parse();
    Log("main(): start");

    clientlist := list.New();
    in := make(chan string);
    Log("main(): start handlingINOUT()");
    go handlingINOUT(in, clientlist);

    netlisten, err := net.Listen("tcp", "0.0.0.0:6667");
    test(err, "main Listen");
    defer netlisten.Close();

    for {
        // wait for clients
        Log("main(): wait for client ...");
        conn, err := netlisten.Accept();
        test(err, "main: Accept for client");
        go clientHandling(conn, in, clientlist);
    }
}
