import React from 'react';

export default function SidebarRight() {
    return (
        <div style={{padding: '20px'}}>
            <h3>🎙️ Talking Stick</h3>
            <div style={{marginTop: '20px'}}>
                <h4>Current Speaker</h4>
                <p style={{color: '#888'}}>No one has the floor</p>
            </div>
            <div style={{marginTop: '20px'}}>
                <h4>Queue</h4>
                <p style={{color: '#888'}}>No one waiting</p>
            </div>
            <div style={{marginTop: '20px'}}>
                <p style={{fontSize: '12px', color: '#666'}}>
                    Facilitator controls coming soon...
                </p>
            </div>
        </div>
    );
}
