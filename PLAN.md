## General idea

I want to create a website that displays trending World of Warcraft addons for **Retail** version. Data should be synced and saved hourly from the CurseForge API. Historical download snapshots are stored to calculate velocity-based trending scores

## Main focus

Main focus of this app should be to create an unique and comprehensive trendiness algorithm for showing up addons.

## Trending Algorithm

This is my MVP idea of trending algorithm which I'd like to improve. This is the main goal of this project so a special attention and research should be given when coming up with a proper solution. 

**Hot Right Now** - Top addons by absolute download velocity
- Requires 500+ total downloads
- Sorted by downloads gained in the last 24 hours
- Shows established addons with high activity

**Rising Stars** - Top addons by percentage growth
- Requires 50-5,000 total downloads
- Sorted by growth percentage (7-day preferred, 24-hour fallback)
- Shows smaller addons gaining traction

**Tier Multipliers** - To prevent tiny addons from dominating:
- 0-10 downloads: 0.1x
- 11-50: 0.3x
- 51-100: 0.5x
- 101-500: 0.7x
- 501-1000: 0.85x
- 1001+: 1.0x

## Environment Variables
- `DATABASE_URL` - PostgreSQL connection string
- `CURSEFORGE_API_KEY` - CurseForge API key
- `PORT` - Server port (default: 8080)

## Resources:
- [CurseForge Addons]https://www.curseforge.com/wow
- [CurseForge API Docs](https://docs.curseforge.com/rest-api/)


