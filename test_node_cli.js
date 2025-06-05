// Quick test script to verify node service mounting
import { Wanix } from 'http://localhost:7654/wanix.js';

async function test() {
    console.log('Initializing Wanix...');
    const wanix = await Wanix.init();
    
    console.log('\nChecking /web directory...');
    try {
        const webContents = await wanix.readdir('/web');
        console.log('Contents of /web:', webContents);
        
        if (webContents.includes('node')) {
            console.log('\n✓ SUCCESS: node service is mounted!');
            
            // Check node directory contents
            const nodeContents = await wanix.readdir('/web/node');
            console.log('\nContents of /web/node:', nodeContents);
            
            // Try to read bootstrap.js
            if (nodeContents.includes('bootstrap.js')) {
                const content = await wanix.readFile('/web/node/bootstrap.js', 'utf8');
                console.log('\n✓ bootstrap.js exists, size:', content.length, 'bytes');
                console.log('First line:', content.split('\n')[0]);
            }
        } else {
            console.log('\n✗ FAILED: node service NOT found in /web');
        }
    } catch (e) {
        console.error('\nERROR:', e.message);
    }
}

test().catch(console.error);