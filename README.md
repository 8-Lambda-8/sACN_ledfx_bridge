# sACN ledfx Bridge

Select scenes by sACN values

## Installing

```
git clone https://github.com/8-Lambda-8/sACN_ledfx_bridge
cd sACN_ledfx_bridge
npm i
```

## Configuration

change the config.json to your needs

the scenes array holds the scene IDs from ledFx (currently max 255)

```
{
  "sAcnUniverse": 1,
  "channel": 1,
  "scenes": ["scene1", "scene2"],
  "ledfx_host": "127.0.0.1",
  "ledfx_port": "8888"
}
```

## Usage

```
npm run start
```

the DMX Values 1-255 activate the scenes

Value 0 deactivates the last activated scene
