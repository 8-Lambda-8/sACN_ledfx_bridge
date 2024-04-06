import { Server } from "e131";
// config
const sAcnUniverse = 1;
const channel = 0;

const server = new Server([sAcnUniverse]);

server.on("listening", () => console.log("Started Listening"));

server.on("error", console.error);
server.on("packet-error", console.warn);
server.on("packet-out-of-order", console.warn);

server.on("packet", (inPacket: any) => {
  let inSlotsData = inPacket.getSlotsData() as Buffer;
});
