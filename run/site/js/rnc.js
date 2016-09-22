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
	this.player_index = -1;
	this.zoom = 1;

	// users
	this.userid = undefined;
	this.users_turn = undefined;
	
	// game objects
	this.players = [];
	this.team_color = ["#03c", "#c00", "#60c", "#093", "#0cc", "#fc0"]; // 6 starter color, we will randomly add more as needed (only multiples of 3)
	this.objects = [];

	this.ws = new WebSocket(url);
	that = this;
	this.ws.onmessage = function(evt) {
		var msg = JSON.parse(evt.data);
		that.handle_message(msg.message_type, msg.data);
	};

	// Set up interaction
	this.canvas.addEventListener('wheel', function(evt){
		that.zoom += evt.deltaY / 30;
		if (that.zoom <= 1) {
			that.zoom = 1;
		}
		that.draw();
		return false; 
	}, false);
	this.canvas.addEventListener('mousedown', function(evt){
		var rect = that.canvas.getBoundingClientRect();
		var px = event.clientX - rect.left;
		var py = event.clientY - rect.top;
		coord = that.px2coord(px, py);
		that.handle_click(coord[0], coord[1]);
	});
	this.canvas.addEventListener('mousemove', function(evt){
		var rect = that.canvas.getBoundingClientRect();
		var px = event.clientX - rect.left;
		var py = event.clientY - rect.top;
		coord = that.px2coord(px, py);
		that.handle_point(coord[0], coord[1]);
	});
	document.addEventListener('keydown', function(evt) {
		if (evt.keyCode == 65) {
			that.move_player(-1,0);
		} else if (evt.keyCode == 68) {
			that.move_player(1,0);
		} else if (evt.keyCode == 87) {
			that.move_player(0,-1);
		} else if (evt.keyCode == 83) {
			that.move_player(0,1);
		} else if (evt.keyCode == 32) {
			// next player
			that.next_player();
		} else if (evt.keyCode == 13) {
			console.log('Fire');
		} else if (evt.keyCode == 37) {
			console.log('Aim left');
		} else if (evt.keyCode == 39) {
			console.log('Aim right');
		}
	});
}

RushNCrush.prototype.handle_point = function(x, y) {
	// aim, if it is your turn and you have a guy selected
};

RushNCrush.prototype.handle_click = function(x, y) {
	// shoot if it is your turn and you have a guy selected
};

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

RushNCrush.prototype.move_player = function(dx, dy) {
	if (this.player_index < 0) {
		return;
	}
	i = this.player_index;
	// check if we can move
	if (this.users_turn != this.userid) {
		return;
	}
	p = 0;
	if (this.map[this.players[i].pos.y + dy][this.players[i].pos.x + dx].tType != 8) {
		return;
	}
	// send the move
	this.ws.send("player_move:"+ this.players[i].id +","+ (this.players[i].pos.x + dx) +","+ (this.players[i].pos.y + dy));
};

RushNCrush.prototype.next_player = function() {
	for (var i=1; i<this.players.length; i++) {
		if (this.players[(this.player_index + i) % this.players.length].owner == this.userid) {
			this.player_index = (this.player_index + i) % this.players.length;
			this.focux = this.players[this.player_index].pos.x;
			this.focuy = this.players[this.player_index].pos.y;
			this.draw();
			return;
		}
	}
};

RushNCrush.prototype.update_game = function(data) {
	this.userid = data.your_id;
	this.users_turn = data.current_turn;
	var u_t = data.updated_tiles;
	for (var i=0; i<u_t.length; i++) {
		this.map[u_t.pos.y][u_t.pos.x] = u_t;
	}
	var u_p = data.updated_players;
	for (var i=0; i<u_p.length; i++) {
		found = false;
		for (var j=0; j<this.players.length; j++) {
			if (u_p[i].id == this.players[j].id) {
				this.players[j] = u_p[i];
				if (j == this.player_index) {
					this.focux = u_p[i].pos.x;
					this.focuy = u_p[i].pos.y;
				}
				found = true;
				break;
			}
		}
		if (!found) {
			this.players.push(u_p[i]);
			if (u_p[i].owner == this.userid) {
				// focus on him
				this.focux = u_p[i].pos.x;
				this.focuy = u_p[i].pos.y;
				this.player_index = this.players.length - 1;
			}
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
	
	// Mark things for Shadow
	for (var i=0; i<this.players.length; i++) {
		if (this.players[i].owner == this.userid) {
			this.ray_cast_start(this.players[i].pos.x, this.players[i].pos.y);
		}
	}

	// Draw the tiles
	for (var x=0; x<this.mapw; x++) {
		for (var y=0; y<this.maph; y++) {
			this.draw_tile(this.map[y][x], x, y);
		}
	}

	// Draw the selector icon
	this.draw_cursor(Math.floor(this.focux), Math.floor(this.focuy));

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
	
	px = ((x * this.zoom) - ((this.focux + 0.5) * this.zoom)) + (this.canvas.width / 2);
	py = ((y * this.zoom) - ((this.focuy + 0.5) * this.zoom)) + (this.canvas.height / 2);

	return [px, py];
};

RushNCrush.prototype.px2coord = function(px, py) {
	var x;
	var y;

	x = ((px - (this.canvas.width / 2)) / this.zoom) + (this.focux + 0.5);
	y = ((py - (this.canvas.height / 2)) / this.zoom) + (this.focuy + 0.5);

	return [x,y];
};

RushNCrush.prototype.draw_player = function(player) {
	if (player.pos.x < 0 || player.pos.y < 0) {
		return;
	}
	var center = this.coord2px(player.pos.x + 0.5, player.pos.y + 0.5);
	if (this.team_color[player.owner] == undefined) {
		// add a random color
		color = "";
		for (var i=0; i<99; i++) {
			color = "#" + (Math.floor(Math.random()*5)*3).toString(16) + (Math.floor(Math.random()*5)*3).toString(16) + (Math.floor(Math.random()*5)*3).toString(16);
			repeat = false;
			for (var j=0; j<this.team_color.length; j++) {
				if (color == this.team_color[j]) {
					repeat = true;
					break;
				}
			}
			if (!repeat) {
				break;
			}
		}
		this.team_color[player.owner] = color;
	}
	this.ctx.fillStyle = this.team_color[player.owner];
	this.ctx.beginPath();
	this.ctx.arc(center[0], center[1], this.zoom/2 - 0.5, 0, 2*Math.PI);
	this.ctx.fill();
}

RushNCrush.prototype.draw_tile = function(tile_obj, x, y) {
	no_draw = false;
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
		this.ctx.fillStyle = "#b39b82";
		// Weak vertical cover
		break;
	case 7:
		this.ctx.fillStyle = "#b39b82";
		// Weak horizontal cover
		break;
	case 8:
		no_draw = true;
		// Walkable tile
	case 9:
		no_draw = true;
		// Button
		break;
	}
	
	var topl = this.coord2px(x,y);
	var w = this.zoom;
	var pad = -0.5;
	if (no_draw == false) {
		this.ctx.fillRect(topl[0] + pad, topl[1] + pad, w - pad, w - pad);
	}
	if (!tile_obj.lit) {
		this.ctx.fillStyle = "rgba(0,0,0,0.15)";
		this.ctx.fillRect(topl[0] + pad, topl[1] + pad, w - pad, w - pad);
	}

	// draw debug index
	//this.ctx.fillStyle = "#000000";
	//this.ctx.font="8px";
	//this.ctx.fillText(""+x+","+y, topl[0] + 3, topl[1] + (w/2));
};

RushNCrush.prototype.draw_cursor = function(x, y) {
	this.ctx.strokeStyle = "black";
	this.ctx.lineWidth = this.zoom / 10;
	var topl = this.coord2px(x,y);
	var w = this.zoom;
	var pad = 0;
	this.ctx.strokeRect(topl[0] + pad, topl[1] + pad, w - pad, w - pad);
}


RushNCrush.prototype.ray_cast_start = function(origin_x, origin_y) {
	// clear all the tiles
	for (var x=0; x<this.mapw; x++) {
		for (var y=0; y<this.maph; y++) {
			this.map[y][x].lit = false;
		}
	}
	num_cast = 128;
	for (var i=0; i<num_cast; i++) {
		var sin = Math.sin(Math.PI * 2 * (i / num_cast));
		var cos = Math.cos(Math.PI * 2 * (i / num_cast));
		var ex = cos * 20;
		var ey = sin * 20;
		this.ray_cast(origin_x, origin_y, ex + origin_x, ey + origin_y);
	}	
}

RushNCrush.prototype.ray_cast = function (px, py, ex, ey) {
	var dirx, diry;
	dirx = (ex - px > 0) ? -1 : 1;
	diry = (ey - py > 0) ? -1 : 1;
	var dx = Math.abs(ex - px);
	var dy = Math.abs(ey - py);

	var sx = px;
	var sy = py;
	var n = dx + dy;
	var err = dx - dy;
	dx *= 2;
	dy *= 2;

	for (; n >= 0; n--) {
		if (sx < 0 || sx >= this.mapw || sy < 0 || sy >= this.maph) {
			return;
		}
		this.map[sy][sx].lit = true;
		tt = this.map[sy][sx].tType;
		if (tt == 1 || tt == 2 || tt == 3) {
			return;
		}

		if (err > 0) {
			sx = sx + dirx;
			err = err - dy;
		} else {
			sy = sy + diry;
			err = err + dx;
		}
	}

}


var game = new RushNCrush("ws://"+ document.domain +":12345/", "gamecanvas");
