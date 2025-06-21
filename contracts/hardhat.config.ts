import { vars, HardhatUserConfig } from "hardhat/config";
import "@nomicfoundation/hardhat-toolbox";

const ARB_SEPOLIA_PRIVATE_KEY = vars.get("ARB_SEPOLIA_PRIVATE_KEY");

const config: HardhatUserConfig = {
  solidity: "0.8.28",
  networks: {
    arbSepolia: {
      url: "https://sepolia-rollup.arbitrum.io/rpc",
      accounts: [ARB_SEPOLIA_PRIVATE_KEY],
    },
  },
};

export default config;

