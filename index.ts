import { Server } from "e131";
import blessed from "blessed";

// config
const sAcnUniverse = 1;
const channel = 0;

const server = new Server([sAcnUniverse]);
const screen = blessed.screen();
const line1 = blessed.text({
  top: +screen.height - 5,
  content: "Value|",
  parent: screen,
});

const line2 = blessed.text({
  top: +screen.height - 4,
  content: "[000]|",
  parent: screen,
});

screen.render();

server.on("listening", () => console.log("Started Listening"));

server.on("error", console.error);
server.on("packet-error", console.warn);
server.on("packet-out-of-order", console.warn);

const zeroPad = (num: number, places: number) =>
  String(num).padStart(places, "0");

server.on("packet", (inPacket: any) => {
  let inSlotsData = inPacket.getSlotsData() as Buffer;

  line2.setContent(` ${zeroPad(inSlotsData[channel], 3)} |`);
  screen.render();
});
