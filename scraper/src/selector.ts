import type { Link } from "./api-client";
import { stdin, stdout } from "process";
import { createInterface } from "readline";

/**
 * Display a numbered list of links and prompt user to select one
 */
export async function selectLink(links: Link[]): Promise<Link> {
  if (links.length === 0) {
    throw new Error("No links available to select");
  }

  // Display the list
  console.log(`\nFound ${links.length} link(s):\n`);
  links.forEach((link, index) => {
    const num = index + 1;
    console.log(`${num}. ${link.url}`);
    if (link.title) {
      console.log(`   Title: ${link.title}`);
    } else {
      console.log(`   Title: (no title)`);
    }
    console.log("");
  });

  // Prompt for selection
  const rl = createInterface({
    input: stdin,
    output: stdout,
  });

  return new Promise((resolve, reject) => {
    const askForSelection = () => {
      rl.question(
        `Select a link to scrape (1-${links.length}): `,
        (answer) => {
          const selection = parseInt(answer.trim(), 10);

          if (isNaN(selection) || selection < 1 || selection > links.length) {
            console.error(
              `Invalid selection. Please enter a number between 1 and ${links.length}.`
            );
            askForSelection();
            return;
          }

          rl.close();
          resolve(links[selection - 1]);
        }
      );
    };

    askForSelection();

    // Handle Ctrl+C
    rl.on("SIGINT", () => {
      rl.close();
      reject(new Error("Selection cancelled"));
    });
  });
}

// method to ask user if they would like to update the link
export async function askToUpdateLink(): Promise<boolean> {
  const rl = createInterface({
    input: stdin,
    output: stdout,
  });

  return new Promise((resolve) => {
    rl.question(`Would you like to update the link? (y/n) `, (answer) => {
      resolve(answer.trim() === "y");
    });
  });
}

// method to ask user if they would like to force update the link
export async function askToForceUpdateLink(): Promise<boolean> {
  const rl = createInterface({
    input: stdin,
    output: stdout,
  });

  return new Promise((resolve) => {
    rl.question(`Would you like to force update the link? (y/n) `, (answer) => {
      resolve(answer.trim() === "y");
    });
  });
}
