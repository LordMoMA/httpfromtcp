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
