const express = require('express');
const path = require('path');

const app = express();
const PORT = process.env.PORT || 3000;

// Set EJS as template engine
app.set('view engine', 'ejs');
app.set('views', path.join(__dirname, 'views'));

// Serve static files
app.use(express.static(path.join(__dirname, 'public')));

// Routes
app.get('/', (req, res) => {
    res.render('overview', {
        title: 'Cosmos Rewards Overview',
        currentPage: 'overview'
    });
});

app.get('/calculator', (req, res) => {
    res.render('calculator', {
        title: 'Rewards Calculator',
        currentPage: 'calculator'
    });
});

app.get('/code-references', (req, res) => {
    res.render('code-references', {
        title: 'Code References',
        currentPage: 'code-references'
    });
});

// Start server
app.listen(PORT, () => {
    console.log(`🚀 Cosmos Rewards Calculator running at http://localhost:${PORT}`);
    console.log('📊 Educational tool for understanding Cosmos blockchain inflation and rewards');
    console.log('🌐 Open your browser and go to: http://localhost:3000');
});

// Graceful shutdown
process.on('SIGINT', () => {
    console.log('\n👋 Shutting down Cosmos Rewards Calculator...');
    process.exit(0);
});