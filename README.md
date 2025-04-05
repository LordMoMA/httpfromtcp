# How to Run the Project

```bash
# in one terminal
go run ./cmd/tcplistener | tee /tmp/requestline.txt

# in another one
curl http://localhost:42069/david/lee
```

# Goroutines and Server Architecture

## Why use goroutines?

### go server.listen() in Serve():

This runs the server's listening loop in a background goroutine

Allows Serve() to return immediately rather than blocking

Makes it possible to start the server and still use the main thread for other tasks (like waiting for shutdown signals)

### go s.handle(conn) in listen():

Creates a new goroutine for each incoming connection
Enables the server to handle multiple connections concurrently
Prevents one slow client from blocking others

# Method Hierarchy

The methods have a clear hierarchy:

## Serve(port): Top-level function

Creates the server instance
Sets up the TCP listener
Initializes the server state
Starts the listening goroutine
Returns control to caller

## listen(): Mid-level method

Runs in its own goroutine
Accepts new connections in a loop
Spawns a handler goroutine for each new connection
Continues accepting connections until server is stopped

## handle(conn): Low-level method

Runs in its own goroutine for each client
Processes a single HTTP request
Generates the appropriate response
Closes the connection when finished

## Response helpers: Utility methods

writeSuccessResponse()
writeErrorResponse()



# Why Use a Goroutine Here?

You're right that the file must be read sequentially—we can't process data before reading it. The goroutine here doesn't mean things happen out of order; it means:

## Benefits of Using a Goroutine

- **Non-blocking return**: The `getLinesChannel` function returns immediately with a channel, rather than waiting for the entire file to be processed.
- **Concurrent processing**: The `main` function can start consuming lines while the goroutine continues reading the file.
- **Decoupling**: The producer (file reader) and consumer (line printer) can work at their own pace.

## No Risk of Disordering

The data cannot be processed out of order because:

- The goroutine reads the file sequentially, byte by byte.
- It only sends complete lines to the channel when they're ready.
- Channels in Go preserve the order of sent messages.
- The `main` function processes the lines in the order they arrive on the channel.

## What's Async Here?

There are two operations happening concurrently:

1. **Reading/parsing**: The goroutine reads chunks from the file and parses them into lines.
2. **Processing/printing**: The `main` function prints lines as they become available.

## Benefits of This Pattern

- **Memory efficiency**: You don't need to read the entire file into memory before processing.
- **Responsiveness**: The program can start producing output before the entire input is read.
- **Throughput**: In a more complex program, you could have multiple consumers processing the data.
- **Resource utilization**: The CPU can work on printing while waiting for I/O operations.

## Real-World Analogy

Think of it like an assembly line:

- **Worker 1 (goroutine)**: Reads chunks and assembles complete lines.
- **Conveyor belt (channel)**: Transports complete lines.
- **Worker 2 (main function)**: Takes lines and prints them.

Each worker does their job concurrently, but the processing order is maintained.

## Use Sequential When:

```go
func getLines(f io.ReadCloser) []string {
    defer f.Close()
    
    var lines []string
    data := make([]byte, 8)
    currentLine := ""
    
    for {
        count, err := f.Read(data)
        if err == io.EOF {
            if currentLine != "" {
                lines = append(lines, currentLine)
            }
            break
        }
        if err != nil {
            fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
            break
        }
        
        chunk := string(data[:count])
        parts := strings.Split(chunk, "\n")
        
        for i := 0; i < len(parts)-1; i++ {
            currentLine += parts[i]
            lines = append(lines, currentLine)
            currentLine = ""
        }
        
        currentLine += parts[len(parts)-1]
    }
    
    return lines
}

func main() {
    file, err := os.Open("messages.txt")
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error opening file: %v\n", err)
        os.Exit(1)
    }
    
    lines := getLines(file)
    
    for _, line := range lines {
        fmt.Printf("read: %s\n", line)
    }
}
```

### Files are small

Code simplicity is a priority
Complete data is needed before processing can begin

## Use Goroutines/Channels When:

### Files are large

Processing can start with partial data
You want to utilize IO waiting time
You need to process streams (like network connections)
You're building a pipeline of operations

## For UDP

```bash
# in one terminal
go run ./cmd/udpsender

# in another terminal
nc -ul 42069
```

### Reader Inside vs. Outside the Loop

```go
// Option 1: Create reader each time
for {
    line, err := bufio.NewReader(os.Stdin).ReadString('\n')
    // ...
}

// Option 2: Create reader once
reader := bufio.NewReader(os.Stdin)
for {
    line, err := reader.ReadString('\n')
    // ...
}
```



### Memory allocation: 

Option 1 creates a new reader on every loop iteration, which means:

- New memory allocation each time
- More work for garbage collection
- Slightly higher CPU usage

### Buffer reuse: 

Option 2 reuses the same reader, which means:

The internal buffer gets reused
Less memory churn
More efficient

### State: 

The reader maintains internal state about its buffer. When created outside the loop:

- It can use information from previous reads to optimize future reads
- Better handling of partial reads (if a read doesn't consume all data)
