# sACN ledfx Bridge

Select scenes by sACN values

## Rewritten in GO
for old nodejs version go to [nodejs branch](https://github.com/8-Lambda-8/sACN_ledfx_bridge/tree/nodejs)

## New TUI
![image](https://github.com/user-attachments/assets/ce494616-2060-41cf-95fc-bc634cd4999f)

### Example configuration for QLC+:
- QLC+, LedFx and sACN_ledfx_bridge running on same machine
- QLC:
  - 127.0.0.1 network
  - Multicast: off
  - Port: 5568 (Default)
  - E1.31 Universe 1 (Default)
  - Transmission Mode: Full (Default)
  - Priority: 100 (Default)
- Bridge:
  - Universe: 1 (Default)
  - Channel: 1 (Default)
  - LedFx Host: http://127.0.0.1:8888 (Default)
  - Scenes: go into scene submenu and select "get scenes from LedFx" to load all scenes



![image](https://github.com/user-attachments/assets/27a1b6d9-208f-4606-9000-ac30cd6a63e1)
