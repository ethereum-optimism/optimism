// SPDX-License-Identifier: MIT
pragma solidity >0.5.0 <0.8.0;

library Lib_RingBuffer {
    using Lib_RingBuffer for RingBuffer;

    /***********
     * Structs *
     ***********/

    struct Buffer {
        uint256 length;
        mapping (uint256 => bytes32) buf;
    }

    struct RingBuffer {
        bytes32 contextA;
        bytes32 contextB;
        Buffer bufferA;
        Buffer bufferB;
        uint256 nextOverwritableIndex;
    }

    struct RingBufferContext {
        // contextA
        uint40 globalIndex;
        bytes27 extraData;

        // contextB
        uint64 currBufferIndex;
        uint40 prevResetIndex;
        uint40 currResetIndex;
    }


    /*************
     * Constants *
     *************/

    uint256 constant MIN_CAPACITY = 16;


    /**********************
     * Internal Functions *
     **********************/

    /**
     * Pushes a single element to the buffer.
     * @param _self Buffer to access.
     * @param _value Value to push to the buffer.
     * @param _extraData Optional global extra data.
     */
    function push(
        RingBuffer storage _self,
        bytes32 _value,
        bytes27 _extraData
    )
        internal
    {
        RingBufferContext memory ctx = _self.getContext();
        Buffer storage currBuffer = _self.getBuffer(ctx.currBufferIndex);

        // Set a minimum capacity.
        if (currBuffer.length == 0) {
            currBuffer.length = MIN_CAPACITY;
        }

        // Check if we need to expand the buffer.
        if (ctx.globalIndex - ctx.currResetIndex >= currBuffer.length) {
            if (ctx.currResetIndex < _self.nextOverwritableIndex) {
                // We're going to overwrite the inactive buffer.
                // Bump the buffer index, reset the delete offset, and set our reset indices.
                ctx.currBufferIndex++;
                ctx.prevResetIndex = ctx.currResetIndex;
                ctx.currResetIndex = ctx.globalIndex;

                // Swap over to the next buffer.
                currBuffer = _self.getBuffer(ctx.currBufferIndex);
            } else {
                // We're not overwriting yet, double the length of the current buffer.
                currBuffer.length *= 2;
            }
        }

        // Index to write to is the difference of the global and reset indices.
        uint256 writeHead = ctx.globalIndex - ctx.currResetIndex;
        currBuffer.buf[writeHead] = _value;

        // Bump the global index and insert our extra data, then save the context.
        ctx.globalIndex++;
        ctx.extraData = _extraData;
        _self.setContext(ctx);
    }

    /**
     * Pushes a single element to the buffer.
     * @param _self Buffer to access.
     * @param _value Value to push to the buffer.
     */
    function push(
        RingBuffer storage _self,
        bytes32 _value
    )
        internal
    {
        RingBufferContext memory ctx = _self.getContext();
        
        _self.push(
            _value,
            ctx.extraData
        );
    }

    /**
     * Pushes a two elements to the buffer.
     * @param _self Buffer to access.
     * @param _valueA First value to push to the buffer.
     * @param _valueA Second value to push to the buffer.
     * @param _extraData Optional global extra data.
     */
    function push2(
        RingBuffer storage _self,
        bytes32 _valueA,
        bytes32 _valueB,
        bytes27 _extraData
    )
        internal
    {
        _self.push(_valueA, _extraData);
        _self.push(_valueB, _extraData);
    }

    /**
     * Pushes a two elements to the buffer.
     * @param _self Buffer to access.
     * @param _valueA First value to push to the buffer.
     * @param _valueA Second value to push to the buffer.
     */
    function push2(
        RingBuffer storage _self,
        bytes32 _valueA,
        bytes32 _valueB
    )
        internal
    {
        RingBufferContext memory ctx = _self.getContext();

        _self.push2(
            _valueA,
            _valueB,
            ctx.extraData
        );
    }

    /**
     * Retrieves an element from the buffer.
     * @param _self Buffer to access.
     * @param _index Element index to retrieve.
     * @return Value of the element at the given index.
     */
    function get(
        RingBuffer storage _self,
        uint256 _index
    )
        internal
        view
        returns (
            bytes32    
        )
    {
        RingBufferContext memory ctx = _self.getContext();

        require(
            _index < ctx.globalIndex,
            "Index out of bounds."
        );

        Buffer storage currBuffer = _self.getBuffer(ctx.currBufferIndex);
        Buffer storage prevBuffer = _self.getBuffer(ctx.currBufferIndex + 1);

        if (_index >= ctx.currResetIndex) {
            // We're trying to load an element from the current buffer.
            // Relative index is just the difference from the reset index.
            uint256 relativeIndex = _index - ctx.currResetIndex;

            // Shouldn't happen but why not check.
            require(
                relativeIndex < currBuffer.length,
                "Index out of bounds."
            );

            return currBuffer.buf[relativeIndex];
        } else {
            // We're trying to load an element from the previous buffer.
            // Relative index is the difference from the reset index in the other direction.
            uint256 relativeIndex = ctx.currResetIndex - _index;

            // Condition only fails in the case that we deleted and flipped buffers.
            require(
                ctx.currResetIndex > ctx.prevResetIndex,
                "Index out of bounds."
            );

            // Make sure we're not trying to read beyond the array.
            require(
                relativeIndex <= prevBuffer.length,
                "Index out of bounds."
            );

            return prevBuffer.buf[prevBuffer.length - relativeIndex];
        }
    }

    /**
     * Deletes all elements after (and including) a given index.
     * @param _self Buffer to access.
     * @param _index Index of the element to delete from (inclusive).
     * @param _extraData Optional global extra data.
     */
    function deleteElementsAfterInclusive(
        RingBuffer storage _self,
        uint40 _index,
        bytes27 _extraData
    )
        internal
    {
        RingBufferContext memory ctx = _self.getContext();

        require(
            _index < ctx.globalIndex && _index >= ctx.prevResetIndex,
            "Index out of bounds."
        );

        Buffer storage currBuffer = _self.getBuffer(ctx.currBufferIndex);
        Buffer storage prevBuffer = _self.getBuffer(ctx.currBufferIndex + 1);

        if (_index < ctx.currResetIndex) {
            // We're switching back to the previous buffer.
            // Reduce the buffer index, set the current reset index back to match the previous one.
            // We use the equality of these two values to prevent reading beyond this buffer.
            ctx.currBufferIndex--;
            ctx.currResetIndex = ctx.prevResetIndex;
        }

        // Set our global index and extra data, save the context.
        ctx.globalIndex = _index;
        ctx.extraData = _extraData;
        _self.setContext(ctx);
    }

    /**
     * Deletes all elements after (and including) a given index.
     * @param _self Buffer to access.
     * @param _index Index of the element to delete from (inclusive).
     */
    function deleteElementsAfterInclusive(
        RingBuffer storage _self,
        uint40 _index
    )
        internal
    {
        RingBufferContext memory ctx = _self.getContext();
        _self.deleteElementsAfterInclusive(
            _index,
            ctx.extraData
        );
    }

    /**
     * Retrieves the current global index.
     * @param _self Buffer to access.
     * @return Current global index.
     */
    function getLength(
        RingBuffer storage _self
    )
        internal
        view
        returns (
            uint40
        )
    {
        RingBufferContext memory ctx = _self.getContext();
        return ctx.globalIndex;
    }

    /**
     * Changes current global extra data.
     * @param _self Buffer to access.
     * @param _extraData New global extra data.
     */
    function setExtraData(
        RingBuffer storage _self,
        bytes27 _extraData
    )
        internal
    {
        RingBufferContext memory ctx = _self.getContext();
        ctx.extraData = _extraData;
        _self.setContext(ctx);
    }

    /**
     * Retrieves the current global extra data.
     * @param _self Buffer to access.
     * @return Current global extra data.
     */
    function getExtraData(
        RingBuffer storage _self
    )
        internal
        view
        returns (
            bytes27
        )
    {
        RingBufferContext memory ctx = _self.getContext();
        return ctx.extraData;
    }

    /**
     * Sets the current ring buffer context.
     * @param _self Buffer to access.
     * @param _ctx Current ring buffer context.
     */
    function setContext(
        RingBuffer storage _self,
        RingBufferContext memory _ctx
    )
        internal
        returns (
            bytes32
        )
    {
        bytes32 contextA;
        bytes32 contextB;

        uint40 globalIndex = _ctx.globalIndex;
        bytes27 extraData = _ctx.extraData;
        assembly {
            contextA := globalIndex
            contextA := or(contextA, extraData)
        }

        uint64 currBufferIndex = _ctx.currBufferIndex;
        uint40 prevResetIndex = _ctx.prevResetIndex;
        uint40 currResetIndex = _ctx.currResetIndex;
        assembly {
            contextB := currBufferIndex
            contextB := or(contextB, shl(64, prevResetIndex))
            contextB := or(contextB, shl(104, currResetIndex))
        }

        if (_self.contextA != contextA) {
            _self.contextA = contextA;
        }

        if (_self.contextB != contextB) {
            _self.contextB = contextB;
        }
    }

    /**
     * Retrieves the current ring buffer context.
     * @param _self Buffer to access.
     * @return Current ring buffer context.
     */
    function getContext(
        RingBuffer storage _self
    )
        internal
        view
        returns (
            RingBufferContext memory
        )
    {
        bytes32 contextA = _self.contextA;
        bytes32 contextB = _self.contextB;

        uint40 globalIndex;
        bytes27 extraData;
        assembly {
            globalIndex := and(contextA, 0x000000000000000000000000000000000000000000000000000000FFFFFFFFFF)
            extraData   := and(contextA, 0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF0000000000)
        }

        uint64 currBufferIndex;
        uint40 prevResetIndex;
        uint40 currResetIndex;
        assembly {
            currBufferIndex :=          and(contextB, 0x000000000000000000000000000000000000000000000000FFFFFFFFFFFFFFFF)
            prevResetIndex  := shr(64,  and(contextB, 0x00000000000000000000000000000000000000FFFFFFFFFF0000000000000000))
            currResetIndex  := shr(104, and(contextB, 0x0000000000000000000000000000FFFFFFFFFF00000000000000000000000000))
        }

        return RingBufferContext({
            globalIndex: globalIndex,
            extraData: extraData,
            currBufferIndex: currBufferIndex,
            prevResetIndex: prevResetIndex,
            currResetIndex: currResetIndex
        });
    }

    /**
     * Retrieves the a buffer from the ring buffer by index.
     * @param _self Buffer to access.
     * @param _which Index of the sub buffer to access.
     * @return Sub buffer for the index.
     */
    function getBuffer(
        RingBuffer storage _self,
        uint256 _which
    )
        internal
        view
        returns (
            Buffer storage
        )
    {
        return _which % 2 == 0 ? _self.bufferA : _self.bufferB;
    }
}
