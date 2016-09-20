// Rush 'n Crush example client

function RushNCrush(url, canvas_id) {
	this.canvas = document.getElementById(canvas_id);
	this.ctx = this.canvas.getContext("2d");

	// map variables
	this.map = [[]];
	this.mapw = 0;
	this.maph = 0;
	this.focux = 0;
	this.focuy = 0;
	this.zoom = 1;

	// users
	this.userid = undefined;
	this.users_turn = undefined;
	
	// game objects
	this.players = [];
	this.team_color = ["#ff0000","#00ff00", "#0000ff"];
	this.objects = [];

	this.ws = new WebSocket(url);
	that = this;
	this.ws.onmessage = function(evt) {
		var msg = JSON.parse(evt.data);
		that.handle_message(msg.message_type, msg.data);
	};
}

RushNCrush.prototype.handle_message = function(message_type, data) {
	changed = false;
	if (message_type == "map") {
		changed = this.build_map(data)
	} else if (message_type == "update") {
		changed = this.update_game(data)	
	}

	if (changed == true) {
		this.draw();
	}
};

RushNCrush.prototype.update_game = function(data) {
	this.userid = data.your_id
	var u_t = data.updated_tiles;
	for (var i=0; i<u_t.length; i++) {
		this.map[u_t.pos.y][u_t.pos.x] = u_t;
	}
	var u_p = data.updated_players;
	for (var i=0; i<u_p.length; i++) {
		found = false;
		for (var j=0; j<this.players.length; j++) {
			if (u_p[i].id == this.players[j].id) {
				this.players[j] = u_p[i]
				found = true;
				break;
			}
		}
		if (!found) {
			this.players.push(u_p[i]);
		}
	}
	return true;
};

RushNCrush.prototype.build_map = function(maparr) {
	this.map = maparr;
	this.maph = maparr.length;
	this.mapw = maparr[0].length;	

	// set default focus
	this.focux = this.mapw / 2.0;
	this.focuy = this.maph / 2.0;

	// set default zoom
	this.zoom = this.canvas.width / this.mapw;
	if (this.canvas.height / this.maph < this.zoom) {
		this.zoom = this.canvas.height / this.maph;
	}

	return true;
};

RushNCrush.prototype.draw = function() {
	// Clear
	this.ctx.clearRect(0, 0, this.canvas.width, this.canvas.height);
	
	// Draw the tiles
	for (var x=0; x<this.mapw; x++) {
		for (var y=0; y<this.maph; y++) {
			this.draw_tile(this.map[y][x], x, y);
		}
	}

	// Draw the players
	for (var i=0; i<this.players.length; i++) {
		this.draw_player(this.players[i]);
	}
};

// Turns an x,y coordinate to the pixel coordinate at the top left of the box
// returns a array with [px, py]
RushNCrush.prototype.coord2px = function(x, y) {
	var px;
	var py;
	
	px = ((x * this.zoom) - (this.focux * this.zoom)) + (this.canvas.width / 2);
	py = ((y * this.zoom) - (this.focuy * this.zoom)) + (this.canvas.height / 2);

	return [px,py];
};

RushNCrush.prototype.draw_player = function(player) {
	var center = this.coord2px(player.pos.x + 0.5, player.pos.y + 0.5);
	this.ctx.fillStyle = this.team_color[player.owner];
	this.ctx.beginPath();
	this.ctx.arc(center[0], center[1], this.zoom/2, 0, 2*Math.PI);
	this.ctx.fill();
}

RushNCrush.prototype.draw_tile = function(tile_obj, x, y) {
	switch (tile_obj.tType) {
	case 1:
		this.ctx.fillStyle = "#000000";
		// Invincible wall
		break;
	case 2:
		this.ctx.fillStyle = "#5c5c5c";
		// Strong wall
		break;
	case 3:
		this.ctx.fillStyle = "#4a301b";
		// Weak wall
		break;
	case 4:
		this.ctx.fillStyle = "#b0b0b0";
		// Strong vertical cover
		break;
	case 5:
		this.ctx.fillStyle = "#b0b0b0";
		// Strong horizontal cover
		break;
	case 6:
		this.ctx.fillStyle = "#685954";
		// Weak vertical cover
		break;
	case 7:
		this.ctx.fillStyle = "#685954";
		// Weak horizontal cover
		break;
	case 8:
		return;
		// Walkable tile
		break;
	case 9:
		// Button
		break;
	}
	var topl = this.coord2px(x,y);
	var w = this.zoom;
	this.ctx.fillRect(topl[0], topl[1], w, w);
};

var game = new RushNCrush("ws://"+ document.domain +":12345/", "gamecanvas");
