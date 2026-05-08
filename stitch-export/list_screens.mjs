import { stitch } from '@google/stitch-sdk';

const projectId = '7371947775132871122';
const wanted = new Set([
  'b47bc6d4b17d41f49040efc4c790edaa',
  '792b5386be2e418ebb11961eec12d883',
  'd527aa6030db4eb0ab0440d8aec0d190',
  '67417f5e41e8449199c9a5fb5832241a',
  '6c59bb4a7f3c4975a6f5b521acc0a63d',
  '8b75d11e0b354022b87f4ccdb48a726c',
  'c009797430504244b0f984b4a52d5f7f',
  'dbfaa96e02a5447d80244f0e5a2b8976',
  'c1f0c00e60d54bf2a0ccbdb326b790d2',
  '6bdc2e36f6f241928f856c46076ed762',
  'ca19c5c61e364811a3c6bda6ea2c3c03',
]);

const project = stitch.project(projectId);
const screens = await project.screens();

const result = [];
for (const screen of screens) {
  if (!wanted.has(screen.id)) continue;
  const imageUrl = await screen.getImage();
  const htmlUrl = await screen.getHtml();
  result.push({ id: screen.id, imageUrl, htmlUrl });
}

console.log(JSON.stringify({ count: result.length, screens: result }, null, 2));
