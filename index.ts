import http from "http";
import { Server } from "e131";
import blessed from "blessed";

// config
const sAcnUniverse = 1;
const channel = 0;
const scenes = ["baseocean", "orangehigh"];

function activateScene(sceneId: string, deactivate?: boolean) {
  const payload = JSON.stringify({
    id: sceneId,
    action: deactivate ? "deactivate" : "activate",
  });

  const put_options = {
    host: "127.0.0.1",
    port: "8888",
    path: "/api/scenes",
    method: "PUT",
    headers: {
      "Content-Type": "application/json",
    },
  };

  const post_req = http.request(put_options);
  post_req.write(payload);
  post_req.end();
}

const server = new Server([sAcnUniverse]);
const screen = blessed.screen();
const line1 = blessed.text({
  top: +screen.height - 5,
  content: "Value| Scene",
  parent: screen,
});

const line2 = blessed.text({
  top: +screen.height - 4,
  content: "[000]| off",
  parent: screen,
});

screen.render();

server.on("listening", () => console.log("Started Listening"));

server.on("error", console.error);
server.on("packet-error", console.warn);
server.on("packet-out-of-order", console.warn);

const zeroPad = (num: number, places: number) =>
  String(num).padStart(places, "0");

let lastVal = 0;
let scene = "";

server.on("packet", (inPacket: any) => {
  let inSlotsData = inPacket.getSlotsData() as Buffer;

  line2.setContent(` ${zeroPad(inSlotsData[channel], 3)} | ${scene}`);
  screen.render();

  if (inSlotsData[channel] != lastVal) {
    lastVal = inSlotsData[channel];

    if (lastVal == 0) {
      activateScene(scene, true);
      scene = "off";
    }
    if (lastVal - 1 in scenes) {
      scene = scenes[lastVal - 1];
      activateScene(scene);
    }
  }
});
