//go:build amd64
// +build amd64

#include "textflag.h"



// Data in go assemble is in big endian (reverse order)
DATA ·bgraToRgbaMaskSSE+0x00(SB)/4, $0x03000102
DATA ·bgraToRgbaMaskSSE+0x04(SB)/4, $0x07040506
DATA ·bgraToRgbaMaskSSE+0x08(SB)/4, $0x0b08090a
DATA ·bgraToRgbaMaskSSE+0x0c(SB)/4, $0x0f0c0d0e

GLOBL ·bgraToRgbaMaskSSE(SB), RODATA, $16

// func bgraToRgbaInPlaceSSE3(data []byte)
TEXT ·bgraToRgbaInPlaceSSE3(SB), NOSPLIT, $0-24
    MOVQ data_base+0(FP), AX // Load data pointer to the AX register
    MOVQ data_len+8(FP), BX // Load data length to the BX register
    MOVOU ·bgraToRgbaMaskSSE(SB), X1 // Load shuffle mask to register (will be used to convert bgra vectors to rgba)
blockloop:
    CMPQ BX, $16 // Check if the remaining length is less than 16
    JB reduce // If less than 16 - jump to the end


    MOVOU (AX), X0 // Load 32 bytes to the register
    PSHUFB X1, X0 // Shuffle  X0 using mask in X1
    MOVOU X0, (AX) // Store result back to the array


    ADDQ $16, AX // Move array pointer
    SUBQ $16, BX // Reduce how much data left
    JMP blockloop // Back to the next block
reduce:
    CMPQ BX, $4
    JB end

    MOVB (AX), CL        // Load byte from AX+0 into AL (8-bit register)
    MOVB 2(AX), DL       // Load byte from AX+2 into BL (8-bit register)
    MOVB CL, 2(AX)       // Store AL (original AX+0) at AX+2
    MOVB DL, (AX)        // Store BL (original AX+2) at AX+0

    ADDQ $4, AX // Move array pointer
    SUBQ $4, BX // Reduce how much data left
    JMP reduce // Back to the next pixel
end:
    VZEROALL
    RET

