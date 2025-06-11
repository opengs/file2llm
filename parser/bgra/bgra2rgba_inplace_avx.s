//go:build amd64
// +build amd64

#include "textflag.h"



// Data in go assemble is in big endian (reverse order)
DATA ·bgraToRgbaMaskAVX+0x00(SB)/4, $0x03000102
DATA ·bgraToRgbaMaskAVX+0x04(SB)/4, $0x07040506
DATA ·bgraToRgbaMaskAVX+0x08(SB)/4, $0x0b08090a
DATA ·bgraToRgbaMaskAVX+0x0c(SB)/4, $0x0f0c0d0e
DATA ·bgraToRgbaMaskAVX+0x10(SB)/4, $0x13101112
DATA ·bgraToRgbaMaskAVX+0x14(SB)/4, $0x17141516
DATA ·bgraToRgbaMaskAVX+0x18(SB)/4, $0x1b18191a
DATA ·bgraToRgbaMaskAVX+0x1c(SB)/4, $0x1f1c1d1e

GLOBL ·bgraToRgbaMaskAVX(SB), RODATA, $32

// func bgraToRgbaInPlaceAVX2(data []byte)
TEXT ·bgraToRgbaInPlaceAVX2(SB), NOSPLIT, $0-24
    MOVQ data_base+0(FP), AX // Load data pointer to the AX register
    MOVQ data_len+8(FP), BX // Load data length to the BX register
    VMOVDQU ·bgraToRgbaMaskAVX(SB), Y2 // Load shuffle mask to register (will be used to convert bgra vectors to rgba)
blockloop:
    CMPQ BX, $32 // Check if the remaining length is less than 32
    JB reduce // If less than 32 - jump to the end


    VMOVDQU (AX), Y0 // Load 32 bytes to the register
    VPSHUFB Y2, Y0, Y1 // Shuffle  Y0 using mask in Y2 and store data to Y1
    VMOVDQU Y1, (AX) // Store result back to the array
    ADDQ $32, AX // Move array pointer
    SUBQ $32, BX // Reduce how much data left
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

