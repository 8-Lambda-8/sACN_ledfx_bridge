// Type definitions for e131 1.1.3
// Project: github.com/hhromic/e131-node
// Definitions by: 8-Lambda-8 <https://github.com/8-Lambda-8>
// Definitions: https://github.com/DefinitelyTyped/DefinitelyTyped
// Minimum TypeScript Version: 4.2

declare module "e131" {
  /**
   * UDP client for sending E1.31 (sACN) traffic.
   */
  export class Client {
    /**
     * @param arg host address, name or universe number
     * @param port defaults to E1.31 default port 5568
     */
    constructor(arg: string | number, port?: number);

    /**
     * creates a new E1.31 (sACN) packet to be used for sending.
     * */
    createPacket(numSlots: number): Packet;
    /**
     * sends a E1.31 (sACN) packet to the remote host or multicast group.
     */
    send(packet: Packet, callback?: () => void);
  }

  /**
   * A server to handle E131 messages
   */
  export class Server {
    constructor(universes?: number | number[], port?: number[]);
    readonly universes: number[];
    readonly port: number;

    /**
     * fires as soon as the server starts listening
     */
    on(event: "listening", listener: () => void): this;
    /**
     * fires when the server is closed.
     */
    on(event: "close", listener: () => void): this;
    /**
     * fires when an error occurs within the server.
     */
    on(event: "error", listener: () => void): this;
    /**
     * fires when a valid E1.31 (sACN) packet is received.
     */
    on(event: "packet", listener: (packet: Packet) => void): this;
    /**
     * fires when an out-of-order packet is received.
     */
    on(event: "packet-out-of-order", listener: (packet: Packet) => void): this;
    /**
     * fires when an invalid packet is received.
     */
    on(event: "packet-error", listener: (packet: Packet) => void): this;

    /**
     * Closes Server
     */
    close();
  }

  export class Packet {
    readonly Options = {
      TERMINATED = 6,
      PREVIEW = 7,
    };

    constructor(arg: Buffer | number);

    readonly DEFAULT_PRIORITY = 0x64;

    //Setters

    /**
     * sets the CID field into the root layer.
     */
    setCID(uuid: string);
    /**
     * sets source name field into the frame layer.
     */
    setSourceName(name: string);
    /**
     * sets the priority field into the frame layer.
     */
    setPriority(priority: number);
    /**
     * sets the sequence number into the frame layer.
     */
    setSequenceNumber(number: number);
    /**
     * sets the state of a framing option into the frame layer.
     */
    setOption(option: FramingOptions, state: boolean);
    /**
     * sets the DMX universe into the frame layer.
     */
    setUniverse(universe: number);
    /**
     * sets the DMX slots data into the DMP layer
     */
    setSlotsData(buffer: Buffer);

    //Getters

    /**
     * gets the CID field into the root layer.
     */
    getCID(): string;
    /**
     * gets source name field into the frame layer.
     */
    getSourceName(): string;
    /**
     * gets the priority field into the frame layer.
     */
    getPriority(): number;
    /**
     * gets the sequence number into the frame layer.
     */
    getSequenceNumber(): number;
    /**
     * gets the state of a framing option into the frame layer.
     */
    getOption(option: FramingOptions): boolean;
    /**
     * gets the DMX universe into the frame layer.
     */
    getUniverse(): number;
    /**
     * gets the DMX slots data into the DMP layer
     */
    getSlotsData(): Buffer;
  }

  export class e131 {
    DEFAULT_PORT = 5568;

    getMulticastGroup(universe: number): string;
  }
}

declare module "slip";
