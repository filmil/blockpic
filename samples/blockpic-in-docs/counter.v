// SPDX-License-Identifier: Apache-2.0
// A 4-bit counter shown on four LEDs: small enough to draw, with a bus
// that synthesis splits into bits.
module counter (
    input  wire       clk,
    input  wire       reset,
    output reg  [3:0] led
);
    always @(posedge clk) begin
        if (reset)
            led <= 4'b0;
        else
            led <= led + 4'b1;
    end
endmodule
