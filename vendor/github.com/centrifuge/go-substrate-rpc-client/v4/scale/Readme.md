This code was copied over with many thanks from https://github.com/Joystream/parity-codec-go/tree/develop/withreflect

# Chainsafe vs Joystream scale codec

- Code quality wise Chainsafe codec is better because of proper handling(no panic), use of standard interfaces, more tests and good comments.
- However Joystream codec has better usability because of the way it handles structs or unknown types by providing two interfaces.
    ```
    // Encodeable is an interface that defines a custom encoding rules for a data type.
    // Should be defined for structs (not pointers to them).
    // See OptionBool for an example implementation.
    type Encodeable interface {
        // ParityEncode encodes and write this structure into a stream
        ParityEncode(encoder Encoder)
    }

    // Decodeable is an interface that defines a custom encoding rules for a data type.
    // Should be defined for pointers to structs.
    // See OptionBool for an example implementation.
    type Decodeable interface {
        // ParityDecode populates this structure from a stream (overwriting the current contents), return false on failure
        ParityDecode(decoder Decoder)
    }
    ```
    This helped in debugging issues because structs could be debugged one field at a time when decoding and encoding. It was easy to see the progress of decoding for example by looking at already decoded struct fields and the buffere thats passed in within the decoder struct. This could have been implemented in client code for Chainsafe codec in hindsight, but just dealing with RPC execution issues was of higher priority.
    
It's better if the good features of both could be integrated together to create a nicer library.
