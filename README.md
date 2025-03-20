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