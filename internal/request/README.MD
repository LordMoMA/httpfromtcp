## High-Level Overview

The goal of the code is to:

1. Read data incrementally from a stream (e.g., a network connection).

2. Parse the request line (e.g., "GET / HTTP/1.1\r\n") from the data.

3. Store the parsed information (method, target, HTTP version) in the RequestLine struct.

4. Handle cases where the data arrives in chunks (e.g., "GE" in one chunk and "T / HTTP/1.1\r\n" in the next).