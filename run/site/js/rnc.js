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
	this.user_turn = undefined;
	
	// game objects
	this.players = [];
	this.team_color = ["#03c", "#c00", "#60c", "#093", "#0cc", "#fc0"]; // 6 starter color, we will randomly add more as needed (only multiples of 3)
	this.objects = [];

	this.animating = false;
	this.player_ani_queue = [];

	this.ws = new WebSocket(url);
	that = this;
	this.ws.onmessage = function(evt) {
		var msg = JSON.parse(evt.data);
		console.log(msg);
		that.handle_message(msg.message_type, msg.data);
	};

	// Set up interaction
	this.canvas.addEventListener('wheel', function(evt){
		that.zoom += evt.deltaY / 30;
		if (that.zoom <= 1) {
			that.zoom = 1;
		}
		that.draw(true);
		evt.returnValue = false;
		return false; 
	}, false);
	this.canvas.addEventListener('mousedown', function(evt){
		that.handle_click();
	});
	this.canvas.addEventListener('mousemove', function(evt){
		var rect = that.canvas.getBoundingClientRect();
		var px = event.clientX - rect.left;
		var py = event.clientY - rect.top;
		coord = that.px2coord(px, py);
		that.handle_point(coord[0], coord[1]);
	});
	document.addEventListener('keydown', function(evt) {
		if (evt.keyCode == 65 || evt.keyCode == 37) {
			that.move_player(-1,0);
		} else if (evt.keyCode == 68 || evt.keyCode == 39) {
			that.move_player(1,0);
		} else if (evt.keyCode == 87 || evt.keyCode == 38) {
			that.move_player(0,-1);
		} else if (evt.keyCode == 83 || evt.keyCode == 40) {
			that.move_player(0,1);
		} else if (evt.keyCode == 32) {
			that.next_player();
		} else if (evt.keyCode == 13) {
			that.end_turn();
		}
		evt.returnValue = false;
		return false
	});
}

RushNCrush.prototype.handle_point = function(x, y) {
	// aim, if it is your turn and you have a guy selected, and he has turns
	if (this.user_turn != undefined && this.user_turn == this.userid && this.player_index >= 0) {
		px = this.players[this.player_index].pos.x + 0.5;
		py = this.players[this.player_index].pos.y + 0.5;
		ang = 180 * Math.atan2(y - py, x - px) / Math.PI;
		this.players[this.player_index].dir = Math.floor(ang);
		this.draw(false);
	}
};

RushNCrush.prototype.handle_click = function() {
	// shoot if it is your turn and you have a guy selected
	if (this.user_turn == this.userid && this.player_index >= 0) {
		this.ws.send("fire:"+ this.players[this.player_index].id +","+ this.players[this.player_index].weapons[0].name +","+ this.players[this.player_index].dir);
	console.log("sent fire")
	}	
};

RushNCrush.prototype.handle_message = function(message_type, data) {
	changed = false;
	if (message_type == "map") {
		changed = this.build_map(data)
	} else if (message_type == "update") {
		changed = this.update_game(data)	
	}

	if (changed == true) {
		// draw animations
		// then draw everything
		that = this;
		this.run_animations(function() {
			that.draw(true);
		});
	}
};

RushNCrush.prototype.end_turn = function() {
	this.ws.send("end_turn:");
	console.log("sent end_turn")
}

RushNCrush.prototype.move_player = function(dx, dy) {
	if (this.animating) {
		return;
	}
	if (this.player_index < 0) {
		return;
	}
	i = this.player_index;
	// check if we can move
	if (this.user_turn != this.userid) {
		return;
	}
	p = 0;
	if (this.map[this.players[i].pos.y + dy][this.players[i].pos.x + dx].tType != 8) {
		return;
	}
	// send the move
	this.ws.send("player_move:"+ this.players[i].id +","+ (this.players[i].pos.x + dx) +","+ (this.players[i].pos.y + dy) +","+ (this.players[i].dir));
	console.log("sent player_move")
};

RushNCrush.prototype.next_player = function() {
	for (var i=1; i<this.players.length; i++) {
		if (this.players[(this.player_index + i) % this.players.length].owner == this.userid) {
			this.player_index = (this.player_index + i) % this.players.length;
			this.focux = this.players[this.player_index].pos.x;
			this.focuy = this.players[this.player_index].pos.y;
			this.draw(false);
			return;
		}
	}
};

RushNCrush.prototype.update_game = function(data) {
	this.userid = data.your_id;
	this.user_turn = data.current_turn;
	var u_t = data.updated_tiles;
	for (var i=0; i<u_t.length; i++) {
		this.map[u_t[i].pos.y][u_t[i].pos.x] = u_t[i];
	}
	var u_p = data.updated_players;
	if (u_p.length == 0) {
		return true;
	}
	// for every player, if updated, cool, if not, ditch 'em
	for (var p=0; p<this.players.length; p++) {
		p_updated = false;
		for (var i=0; i<u_p.length; i++) {
			if (u_p[i].id == this.players[p].id) {
				// if the player moved, animate it
				if (this.players[p].pos.x != u_p[i].pos.x || this.players[p].pos.y != u_p[i].pos.y) {
					// queue for animation
					this.player_animate(p, this.players[p].pos.x, this.players[p].pos.y, u_p[i].pos.x, u_p[i].pos.y);
				}
				// if we are focusing on this player already, keep the focus
				if (p == this.player_index) {
					this.focux = u_p[i].pos.x;
					this.focuy = u_p[i].pos.y;
				}
				// remove this from our updated array, and break
				this.players[p] = u_p.splice(i,1)[0];
				p_updated = true;
				break;
			}
		}
		if (!p_updated) {
			// remove the player, and adjust
			this.players.splice(p,1);
			// if this player is your focus, reset
			if (p == this.player_index) {
				this.player_index = -1;
				for (var np=0; np < this.players.length; np++) {
					if (this.players[np].owner == this.userid) {
						this.focux = this.players[np].pos.x;
						this.focuy = this.players[np].pos.y;
						this.player_index = np;
						break;
					}
				}
			}
			p--;
		}
	}
	// if there are extras left over, add them
	for (var i=0; i<u_p.length; i++) {
		this.players.push(u_p[i]);
		if (u_p[i].owner == this.userid && this.player_index < 0) {
			// focus on the new player
			this.focux = u_p[i].pos.x;
			this.focuy = u_p[i].pos.y;
			this.player_index = this.players.length - 1;
		}
	}
	// handle hit tiles
	var h_t = data.hit_tiles;
	for (var i=0; i<h_t.length; i++) {
		this.hit_animate(h_t[i].pos.x, h_t[i].pos.y, h_t[i].damage_type);
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

RushNCrush.prototype.run_animations = function(callback) {
	that = this;
	this.animating = true;
	var animate = function(dt) {
		that.draw(false);
		if (that.player_ani_queue.length == 0) {
			that.animating = false;
			callback();
			return;
		}else if (that.player_ani_queue[0]() == false) {
			that.player_ani_queue.shift();
		}
		window.requestAnimationFrame(animate);
	}
	animate();
}

RushNCrush.prototype.hit_animate = function(hitx, hity, type) {
	var steps = 45;
	var anistep = 1;
	var that = this;
	var topl = this.coord2px(hitx,hity);
	var w = this.zoom;
	var draw_hit = function() {
		if (anistep >= steps) {
			return false;
		}
		var fade = 1.0 / anistep;
		var pad = -0.5 + (w * (anistep / (steps * 2)));
		that.ctx.fillStyle = "rgba(200,0,0,"+ fade +")";
		that.ctx.fillRect(topl[0] + pad, topl[1] + pad, w - pad, w - pad);
		anistep++;
		return true;
	}
	this.player_ani_queue.push(draw_hit);
}

RushNCrush.prototype.player_animate = function(p_index, sx,sy, x,y) {
	var steps = 18;
	var anistep = 1;
	var that = this;
	var update_player = function() {
		if (that.players[p_index] === undefined) {
			console.log("Lost one during animation")
			return false
		}
		// change player location and dir
		dx = (x - sx) * (anistep / steps);
		dy = (y - sy) * (anistep / steps);
		that.players[p_index].pos.x = sx + dx;
		that.players[p_index].pos.y = sy + dy;

		// change focus
		if (p_index == that.player_index) {
			that.focux = sx + dx;
			that.focuy = sy + dy;
		}
		anistep = anistep + 1;

		if (anistep > steps) {
			anistep = 0;
			return false
		} else {
			return true
		}
	}
	// put this function into the queue
	this.player_ani_queue.push(update_player);
}

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

RushNCrush.prototype.draw = function(cast) {
	// if the canvas size is wrong, set it
	
	if (cast && (this.ctx.canvas.width != window.innerWidth || this.ctx.canvas.height != window.innerHeight)) {
		this.ctx.canvas.width  = window.innerWidth;
		this.ctx.canvas.height = window.innerHeight;
	}
	// Clear
	this.ctx.clearRect(0, 0, this.canvas.width, this.canvas.height);

	if (cast) {
		this.ray_cast_clear();
		// Mark things for Shadow
		for (var i=0; i<this.players.length; i++) {
			if (this.players[i].owner == this.userid) {
				this.ray_cast_start(this.players[i].pos.x, this.players[i].pos.y);
			}
		}
	}

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

	// Draw the UI
	this.draw_ui();
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
	// draw direction piece
	this.ctx.fillStyle = "#FFFFFF";
	this.ctx.strokeStyle = "#000000";
	this.ctx.lineWidth = this.zoom / 90;
	this.ctx.beginPath();
	this.ctx.moveTo(center[0], center[1]);
	this.ctx.arc(center[0], center[1], this.zoom/2 - 0.5, (Math.PI * player.dir/180) - Math.PI/18, (Math.PI * player.dir/180) + Math.PI/18);
	this.ctx.lineTo(center[0], center[1]);
	this.ctx.fill();
	this.ctx.stroke();
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

RushNCrush.prototype.draw_ui = function() {
	// Draw current Turn
	var box_size = 12;
	var pad = 3;

	this.ctx.fillStyle = this.team_color[this.user_turn];

	var num_boxes = 1;
	if (this.userid == this.user_turn && this.player_index >= 0) {
		num_boxes = this.players[this.player_index].moves;
	}
	for (var i=0; i<num_boxes; i++) {
		var x = pad;
		var y = (i * (pad + box_size)) + pad;

		this.ctx.fillRect(x,y, box_size,box_size);
	}
}

RushNCrush.prototype.ray_cast_clear = function() {
	// clear all the tiles
	for (var x=0; x<this.mapw; x++) {
		for (var y=0; y<this.maph; y++) {
			this.map[y][x].lit = false;
		}
	}
}

RushNCrush.prototype.ray_cast_start = function(origin_x, origin_y) {
	num_cast = 128;
	for (var i=0; i<num_cast; i++) {
		var sin = Math.sin(Math.PI * 2 * (i / num_cast));
		var cos = Math.cos(Math.PI * 2 * (i / num_cast));
		var ex = cos * 20;
		var ey = sin * 20;
		this.ray_cast(origin_x, origin_y, ex + origin_x, ey + origin_y);
	}	
}

RushNCrush.prototype.ray_cast = function(px, py, ex, ey) {
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
